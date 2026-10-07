package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/m-vinc/maco/pkg/engine"
	"github.com/m-vinc/maco/pkg/jobs"
)

// @Summary listNetworks
// @ID listNetworks
// @Description List configured networks and their IDs, modes, and applied topology. Use IDs from this response in VM interface requests. Applied fields describe engine-owned state, not authoring inputs.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags networks
// @Security BearerAuth
// @Produce json
// @Success 200 {array} types.NetworkManifest
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/networks [get]
func (s *Server) listNetworks(w http.ResponseWriter, r *http.Request) {
	networks, err := s.engine.ListNetworks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, networks)
}

// @Summary createNetwork
// @ID createNetwork
// @Description Create a named network. Discover host interface names with listInterfaces. Mode bridge supports members, address (IPv4 CIDR), and vlans; vmnet-bridged requires uplink; vlan requires parent and tag (1 to 4094). Host address does not create DHCP or routing. Use the request schema rather than copying a stored network manifest. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags networks
// @Security BearerAuth
// @Produce json
// @Accept json
// @Param body body engine.CreateNetworkParams true "Request"
// @Success 202 {object} jobs.Job
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/networks [post]
func (s *Server) createNetwork(w http.ResponseWriter, r *http.Request) {
	var params engine.CreateNetworkParams
	if err := readJSON(r, &params); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	s.submitJob(w, r, jobs.Payload{Action: "network.create", Target: params.Name, Network: params})
}

// @Summary applyNetwork
// @ID applyNetwork
// @Description Reconcile the selected network configuration onto the host. Can affect host connectivity; inspect listNetworks and listInterfaces before applying. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags networks
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Success 202 {object} jobs.Job
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/networks/{id}/apply [post]
func (s *Server) applyNetwork(w http.ResponseWriter, r *http.Request) {
	s.submitJob(w, r, jobs.Payload{Action: "network.apply", Target: chi.URLParam(r, "id")})
}

// @Summary destroyNetwork
// @ID destroyNetwork
// @Description Delete a configured network and clean up host resources owned by Maco. May disrupt attached VMs. Borrowed host interfaces are retained. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags networks
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Success 202 {object} jobs.Job
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/networks/{id} [delete]
func (s *Server) destroyNetwork(w http.ResponseWriter, r *http.Request) {
	s.submitJob(w, r, jobs.Payload{Action: "network.destroy", Target: chi.URLParam(r, "id")})
}

// @Summary updateNetwork
// @ID updateNetwork
// @Description Replace the desired settings of an existing network using its ID from listNetworks. Read existing configuration first and supply the desired complete settings. This is not an arbitrary manifest replacement; applied bookkeeping is managed by Maco. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags networks
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Accept json
// @Param body body engine.CreateNetworkParams true "Request"
// @Success 202 {object} jobs.Job
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/networks/{id} [put]
func (s *Server) updateNetwork(w http.ResponseWriter, r *http.Request) {
	var params engine.CreateNetworkParams
	if err := readJSON(r, &params); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	s.submitJob(w, r, jobs.Payload{Action: "network.update", Target: chi.URLParam(r, "id"), Network: params})
}
