package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type CreateSSHKeyRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

func (s *Server) creatorSSHKeys(r *http.Request) ([]string, error) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		return nil, nil
	}

	return s.engine.SSHKeyValues(r.Context(), claims.UserID)
}

// @Summary listSSHKeys
// @ID listSSHKeys
// @Description List the SSH public keys stored on the current account. These keys populate [[ .SSHKeys ]] in guest configurations when a VM is created.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags ssh-keys
// @Security BearerAuth
// @Produce json
// @Success 200 {array} types.SSHKey
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/me/ssh-keys [get]
func (s *Server) listSSHKeys(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	keys, err := s.engine.ListSSHKeys(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, keys)
}

// @Summary addSSHKey
// @ID addSSHKey
// @Description Add an SSH public key to the current account. Supply a name and a complete public key such as ssh-ed25519 AAAA…. Never submit a private key.
// @x-maco {"expose":true,"readOnly":false,"transport":"json"}
// @Tags ssh-keys
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body CreateSSHKeyRequest true "Request"
// @Success 201 {object} types.SSHKey
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/me/ssh-keys [post]
func (s *Server) addSSHKey(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateSSHKeyRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	key, err := s.engine.AddSSHKey(r.Context(), claims.UserID, req.Name, req.PublicKey)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, key)
}

// @Summary deleteSSHKey
// @ID deleteSSHKey
// @Description Delete an SSH public key from the current account by ID from listSSHKeys.
// @x-maco {"expose":true,"readOnly":false,"transport":"json"}
// @Tags ssh-keys
// @Security BearerAuth
// @Produce json
// @Param id path string true "SSH key id"
// @Success 200 {object} StatusResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/me/ssh-keys/{id} [delete]
func (s *Server) deleteSSHKey(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	deleted, err := s.engine.DeleteSSHKey(r.Context(), claims.UserID, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !deleted {
		writeError(w, http.StatusNotFound, "SSH key not found")
		return
	}

	writeJSON(w, http.StatusOK, StatusResponse{Status: "ok"})
}
