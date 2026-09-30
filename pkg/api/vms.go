package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/m-vinc/maco/pkg/engine"
	"github.com/m-vinc/maco/pkg/jobs"
)

// @Summary listVMs
// @ID listVMs
// @Description List all VMs with their manifest, current phase, process ID, and boot time. Use manifest.id as the id argument for VM operations. Read-only; does not start VMs.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Success 200 {array} engine.VMView
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/vms [get]
func (s *Server) listVMs(w http.ResponseWriter, r *http.Request) {
	views, err := s.engine.ListVMs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, views)
}

// @Summary createVM
// @ID createVM
// @Description Create a VM and its disks. Discover image keys and provisioning capabilities with listCatalog, existing media with listMedia, and network IDs with listNetworks first. Body is an API creation request, not a persisted manifest. Provide name, cpus, memory_mib (MiB), disk_size_gib (GiB), and an image key or installation ISO. Example body: {"name":"agent-vm","cpus":2,"memory_mib":2048,"disk_size_gib":20,"username":"maco","image":"<key from listCatalog>"}. The job result contains the new VM ID; creation does not imply the VM is running. Use startVM afterward when needed. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Accept json
// @Param body body engine.CreateVMParams true "Request"
// @Success 202 {object} jobs.Job
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/vms [post]
func (s *Server) createVM(w http.ResponseWriter, r *http.Request) {
	var params engine.CreateVMParams
	if err := readJSON(r, &params); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	keys, err := s.creatorSSHKeys(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.submitJob(w, r, jobs.Payload{Action: "vm.create", Target: params.Name, VM: params, SSHKeys: keys})
}

// @Summary getVM
// @ID getVM
// @Description Inspect one VM by its ID from listVMs, including configuration and current power state. Use this after a completed lifecycle job to verify the result.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} engine.VMView
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/vms/{id} [get]
func (s *Server) getVM(w http.ResponseWriter, r *http.Request) {
	m, st, err := s.engine.StatusVM(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	views, err := s.engine.ListVMs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for _, view := range views {
		if view.Manifest.ID == m.ID {
			writeJSON(w, http.StatusOK, view)
			return
		}
	}

	writeJSON(w, http.StatusOK, engine.VMView{Manifest: m, Phase: string(st.Phase), PID: st.PID})
}

// @Summary startVM
// @ID startVM
// @Description Start an existing VM using its ID from listVMs. Check getVM first and inspect the completed job before assuming the guest is running. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Success 202 {object} jobs.Job
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/vms/{id}/start [post]
func (s *Server) startVM(w http.ResponseWriter, r *http.Request) {
	s.submitJob(w, r, jobs.Payload{Action: "vm.start", Target: chi.URLParam(r, "id")})
}

// @Summary stopVM
// @ID stopVM
// @Description Force-stop a VM using its ID from listVMs. This can interrupt guest writes; prefer shutdownVM for a graceful guest shutdown. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Success 202 {object} jobs.Job
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/vms/{id}/stop [post]
func (s *Server) stopVM(w http.ResponseWriter, r *http.Request) {
	s.submitJob(w, r, jobs.Payload{Action: "vm.stop", Target: chi.URLParam(r, "id")})
}

// @Summary deleteVM
// @ID deleteVM
// @Description Delete a VM and its managed storage. This is destructive. Select the exact VM ID from listVMs and create and verify a backup first if its data must be retained. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Success 202 {object} jobs.Job
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/vms/{id} [delete]
func (s *Server) deleteVM(w http.ResponseWriter, r *http.Request) {
	s.submitJob(w, r, jobs.Payload{Action: "vm.delete", Target: chi.URLParam(r, "id")})
}

// @Summary previewVM
// @ID previewVM
// @Description Read the latest PNG screenshot of a VM display. Returns image content, not JSON; does not send keyboard or pointer input.
// @x-maco {"expose":true,"readOnly":true,"transport":"image"}
// @Tags vms
// @Security BearerAuth
// @Produce png
// @Param id path string true "id"
// @Success 200 {file} binary "PNG preview"
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/vms/{id}/preview [get]
func (s *Server) previewVM(w http.ResponseWriter, r *http.Request) {
	manifest, _, err := s.engine.StatusVM(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	file, err := os.Open(filepath.Join(s.engine.Paths().VMRunDir(manifest.ID), "preview.png"))
	if err != nil {
		writeError(w, http.StatusNotFound, "preview unavailable")
		return
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Preview-Captured-At", strconv.FormatInt(info.ModTime().UnixMilli(), 10))
	w.Header().Set("Content-Type", "image/png")
	http.ServeContent(w, r, "preview.png", info.ModTime(), file)
}

// @Summary updateHardware
// @ID updateHardware
// @Description Update VM hardware settings by ID. Body fields are optional patches: omitted settings remain unchanged. CPU and memory changes require the VM to be stopped; memory_mib uses MiB and disk_size_gib uses GiB. Disk shrinking is unsupported. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Accept json
// @Param body body engine.UpdateHardwareParams true "Request"
// @Success 202 {object} jobs.Job
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/vms/{id}/hardware [patch]
func (s *Server) updateHardware(w http.ResponseWriter, r *http.Request) {
	var params engine.UpdateHardwareParams
	if err := readJSON(r, &params); err != nil {
		writeError(w, http.StatusBadRequest, "invalid hardware settings")
		return
	}

	s.submitJob(w, r, jobs.Payload{Action: "vm.hardware", Target: chi.URLParam(r, "id"), Hardware: params})
}

// @Summary getGuestSetup
// @ID getGuestSetup
// @Description Read guest provisioning settings and image-supported provisioning capabilities for a VM. Consult this before modifying guest setup; secret values may be omitted.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} engine.GuestSetupView
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/vms/{id}/guest-setup [get]
func (s *Server) getGuestSetup(w http.ResponseWriter, r *http.Request) {
	setup, err := s.engine.GuestSetup(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, setup)
}

// @Summary updateGuestSetup
// @ID updateGuestSetup
// @Description Update guest provisioning settings for a stopped VM. Read getGuestSetup first for supported provisioner and mode values. This request is a guest-setup update, not a complete manifest. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Accept json
// @Param body body engine.GuestSetupParams true "Request"
// @Success 202 {object} jobs.Job
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/vms/{id}/guest-setup [patch]
func (s *Server) updateGuestSetup(w http.ResponseWriter, r *http.Request) {
	var params engine.GuestSetupParams
	if err := readJSON(r, &params); err != nil {
		writeError(w, http.StatusBadRequest, "invalid guest setup")
		return
	}

	s.submitJob(w, r, jobs.Payload{Action: "vm.guest-setup", Target: chi.URLParam(r, "id"), GuestSetup: params})
}

// @Summary shutdownVM
// @ID shutdownVM
// @Description Request graceful guest shutdown through ACPI. The guest must support shutdown; inspect getVM after job completion to confirm it stopped. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Success 202 {object} jobs.Job
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/vms/{id}/shutdown [post]
func (s *Server) shutdownVM(w http.ResponseWriter, r *http.Request) {
	s.submitJob(w, r, jobs.Payload{Action: "vm.shutdown", Target: chi.URLParam(r, "id")})
}

// @Summary rebootVM
// @ID rebootVM
// @Description Request a reboot of the selected VM. Use its ID from listVMs and inspect getVM and the completed job for the result. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Success 202 {object} jobs.Job
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/vms/{id}/reboot [post]
func (s *Server) rebootVM(w http.ResponseWriter, r *http.Request) {
	s.submitJob(w, r, jobs.Payload{Action: "vm.reboot", Target: chi.URLParam(r, "id")})
}
