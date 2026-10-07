package api

import (
	"net/http"

	"github.com/m-vinc/maco/pkg/jobs"
)

// @Summary hostInfo
// @ID hostInfo
// @Description Read host hardware, capacity, and runtime information. Use alongside diskStorage and listVMs before planning additional VMs.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags host
// @Security BearerAuth
// @Produce json
// @Success 200 {object} engine.HostInfo
// @Failure 401 {object} ErrorResponse
// @Router /api/host [get]
func (s *Server) hostInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.engine.HostInfo())
}

type HostPowerRequest struct {
	Force bool `json:"force"`
}

// @Summary powerOffHost
// @ID powerOffHost
// @Description Power off the entire Maco host, interrupting API connectivity and running workloads. body.force requests forced handling of running VMs. Job completion may be unobservable once the host shuts down. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags host
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body HostPowerRequest true "Request"
// @Success 202 {object} jobs.Job
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/host/poweroff [post]
func (s *Server) powerOffHost(w http.ResponseWriter, r *http.Request) {
	s.hostPower(w, r, "host.poweroff")
}

// @Summary rebootHost
// @ID rebootHost
// @Description Reboot the entire Maco host, interrupting API connectivity and running workloads. body.force requests forced handling of running VMs. A lost connection during reboot is not proof the reboot failed. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags host
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body HostPowerRequest true "Request"
// @Success 202 {object} jobs.Job
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/host/reboot [post]
func (s *Server) rebootHost(w http.ResponseWriter, r *http.Request) {
	s.hostPower(w, r, "host.reboot")
}

func (s *Server) hostPower(w http.ResponseWriter, r *http.Request, action string) {
	req := HostPowerRequest{}
	if r.ContentLength != 0 {
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request")
			return
		}
	}

	s.submitJob(w, r, jobs.Payload{Action: action, Target: "host", Force: req.Force})
}
