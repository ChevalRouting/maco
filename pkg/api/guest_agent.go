package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/m-vinc/maco/pkg/vm"
)

type guestAgentResponse struct {
	Available bool `json:"available"`
	*vm.GuestAgentInfo
}

// @Summary vmGuestAgent
// @ID vmGuestAgent
// @Description Inspect QEMU guest-agent availability and guest-reported information for a VM ID from listVMs. When available is false the guest agent has not answered yet; this does not mean it is absent or that the VM is stopped. On a freshly started VM the agent commonly takes a while to come up on first boot (Flatcar and similar images ship qemu-ga and start it automatically once the guest finishes booting), so retry a few times over the first minute or two before concluding the agent is unavailable.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} guestAgentResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/vms/{id}/guest-agent [get]
func (s *Server) vmGuestAgent(w http.ResponseWriter, r *http.Request) {
	m, _, err := s.engine.StatusVM(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	info, err := s.engine.GuestAgent(m.ID)
	if err != nil {
		if errors.Is(err, vm.ErrGuestAgentUnavailable) {
			writeJSON(w, http.StatusOK, guestAgentResponse{Available: false})
			return
		}

		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, guestAgentResponse{Available: true, GuestAgentInfo: info})
}
