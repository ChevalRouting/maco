package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/m-vinc/maco/pkg/engine"
	"github.com/m-vinc/maco/pkg/jobs"
	"net/http"
)

// @Summary suggestMAC
// @ID suggestMAC
// @Description Generate a suggested guest MAC address. Use the returned address in a VM interface request when a specific MAC is desired.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Success 200 {object} engine.MACSuggestion
// @Failure 401 {object} ErrorResponse
// @Router /api/interfaces/mac [get]
func (s *Server) suggestMAC(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.engine.SuggestMAC())
}

// @Summary addVMInterface
// @ID addVMInterface
// @Description Add a network interface to a VM. Select the VM ID from listVMs and network ID from listNetworks. Guest addresses use CIDR notation; gateway and nameservers are IP addresses. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Accept json
// @Param body body engine.InterfaceParams true "Request"
// @Success 202 {object} jobs.Job
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/vms/{id}/interfaces [post]
func (s *Server) addVMInterface(w http.ResponseWriter, r *http.Request) {
	s.submitInterface(w, r, "vm.interface.add")
}

// @Summary updateVMInterface
// @ID updateVMInterface
// @Description Update an existing VM interface identified by its interface ID from getVM. Inspect its current settings and the request schema before submitting changes. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Param interface path string true "interface"
// @Accept json
// @Param body body engine.InterfaceParams true "Request"
// @Success 202 {object} jobs.Job
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/vms/{id}/interfaces/{interface} [patch]
func (s *Server) updateVMInterface(w http.ResponseWriter, r *http.Request) {
	s.submitInterface(w, r, "vm.interface.update")
}

// @Summary removeVMInterface
// @ID removeVMInterface
// @Description Remove a VM network interface by its interface ID from getVM. This can disconnect the guest from its network. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Param interface path string true "interface"
// @Success 202 {object} jobs.Job
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/vms/{id}/interfaces/{interface} [delete]
func (s *Server) removeVMInterface(w http.ResponseWriter, r *http.Request) {
	s.submitInterface(w, r, "vm.interface.remove")
}
func (s *Server) submitInterface(w http.ResponseWriter, r *http.Request, action string) {
	var p engine.InterfaceParams
	if action != "vm.interface.remove" {
		if err := readJSON(r, &p); err != nil {
			writeError(w, 400, "invalid interface settings")
			return
		}
	}

	p.ID = chi.URLParam(r, "interface")
	s.submitJob(w, r, jobs.Payload{Action: action, Target: chi.URLParam(r, "id"), Interface: p})
}
