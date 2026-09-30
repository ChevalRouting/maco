package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/m-vinc/maco/pkg/engine"
	"github.com/m-vinc/maco/pkg/jobs"
)

type TemplateRequest struct {
	Name        string                `json:"name" extensions:"x-maco-description=Unique template name. Use an RFC 1123 hostname such as web-server." minLength:"1" maxLength:"63"`
	Description string                `json:"description" binding:"optional" extensions:"x-maco-description=Optional human description of the template."`
	Spec        engine.CreateVMParams `json:"spec" extensions:"x-maco-description=Machine profile: guest provisioning plus hardware defaults. Same shape as createVM without name. The spec name field is ignored."`
}

// @Summary listTemplates
// @ID listTemplates
// @Description List saved machine-profile templates and their IDs. Each template carries guest provisioning (cloud-init or Ignition) and hardware defaults, used to pre-fill VM creation.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags templates
// @Security BearerAuth
// @Produce json
// @Success 200 {array} engine.Template
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/templates [get]
func (s *Server) listTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.engine.ListTemplates()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, templates)
}

// @Summary getTemplate
// @ID getTemplate
// @Description Read a single machine-profile template by ID from listTemplates. The returned spec is a full createVM request without a name, so it can be edited and passed to createVM to customize fields that instantiateTemplate does not override, such as network, addresses, interfaces, disks, or guest_setup.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags templates
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} engine.Template
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/templates/{id} [get]
func (s *Server) getTemplate(w http.ResponseWriter, r *http.Request) {
	t, err := s.engine.GetTemplate(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// @Summary createTemplate
// @ID createTemplate
// @Description Save a reusable machine-profile template. The spec mirrors the createVM request (guest provisioning plus hardware) minus the VM name. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json"}
// @Tags templates
// @Security BearerAuth
// @Produce json
// @Accept json
// @Param body body TemplateRequest true "Request"
// @Success 201 {object} engine.Template
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/templates [post]
func (s *Server) createTemplate(w http.ResponseWriter, r *http.Request) {
	var p TemplateRequest
	if err := readJSON(r, &p); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	t, err := s.engine.CreateTemplate(r.Context(), p.Name, p.Description, p.Spec)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// @Summary updateTemplate
// @ID updateTemplate
// @Description Replace a saved template by ID from listTemplates with the supplied name, description, and spec. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json"}
// @Tags templates
// @Security BearerAuth
// @Produce json
// @Accept json
// @Param id path string true "id"
// @Param body body TemplateRequest true "Request"
// @Success 200 {object} engine.Template
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/templates/{id} [put]
func (s *Server) updateTemplate(w http.ResponseWriter, r *http.Request) {
	var p TemplateRequest
	if err := readJSON(r, &p); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	t, err := s.engine.UpdateTemplate(r.Context(), chi.URLParam(r, "id"), p.Name, p.Description, p.Spec)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// @Summary instantiateTemplate
// @ID instantiateTemplate
// @Description Create a VM from a saved template by ID from listTemplates. This is a convenience path that keeps the template spec as-is and only lets you set a unique VM name and optional cpus, memory_mib, disk_size_gib, or autostart overrides. To change anything else, including networking, addresses, interfaces, disks, or guest provisioning, do not use this endpoint: read the template with getTemplate, edit its spec (a full createVM body without name), and pass it to createVM. Returns a queued job like createVM; the job result contains the new VM ID. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags templates
// @Security BearerAuth
// @Produce json
// @Accept json
// @Param id path string true "id"
// @Param body body engine.TemplateInstanceParams true "Request"
// @Success 202 {object} jobs.Job
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/templates/{id}/instantiate [post]
func (s *Server) instantiateTemplate(w http.ResponseWriter, r *http.Request) {
	var p engine.TemplateInstanceParams
	if err := readJSON(r, &p); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	spec, err := s.engine.TemplateInstance(chi.URLParam(r, "id"), p)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	keys, err := s.creatorSSHKeys(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.submitJob(w, r, jobs.Payload{Action: "vm.create", Target: spec.Name, VM: spec, SSHKeys: keys})
}

// @Summary deleteTemplate
// @ID deleteTemplate
// @Description Delete a saved machine-profile template by ID from listTemplates. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json"}
// @Tags templates
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Success 204 "No content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/templates/{id} [delete]
func (s *Server) deleteTemplate(w http.ResponseWriter, r *http.Request) {
	if err := s.engine.DeleteTemplate(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
