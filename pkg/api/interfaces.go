package api

import "net/http"

// @Summary listInterfaces
// @ID listInterfaces
// @Description List host network interfaces and addresses. Use their actual names for bridge members, VLAN parents, or vmnet uplinks.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags interfaces
// @Security BearerAuth
// @Produce json
// @Success 200 {array} host.Port
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/interfaces [get]
func (s *Server) listInterfaces(w http.ResponseWriter, r *http.Request) {
	ports, err := s.engine.Interfaces()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ports)
}
