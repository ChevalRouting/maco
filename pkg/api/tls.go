package api

import (
	"errors"
	"net/http"

	"github.com/m-vinc/maco/pkg/tlscert"
)

func (s *Server) SetCertificates(certificates *tlscert.Provider) { s.certificates = certificates }

// @Summary getTLS
// @ID getTLS
// @Description Read how the web UI and API certificate is provided (self-signed, custom files, or ACME), the ACME settings, and each served certificate with its issuer and expiry. Private keys and the external account binding key are never returned.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags tls
// @Security BearerAuth
// @Produce json
// @Success 200 {object} tlscert.View
// @Failure 401 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /api/tls [get]
func (s *Server) getTLS(w http.ResponseWriter, _ *http.Request) {
	if s.certificates == nil {
		writeError(w, http.StatusConflict, "tls is disabled")
		return
	}

	writeJSON(w, http.StatusOK, s.certificates.View())
}

// @Summary updateTLS
// @ID updateTLS
// @Description Change how the web UI and API certificate is provided and apply it without a restart. mode is self-signed, file (cert_pem and key_pem; omit both to keep the current files), or acme (directory defaults to Let's Encrypt; domains required; ca_pem trusts a private ACME server; an empty eab_key keeps the stored one). Requires an administrator account.
// @x-maco {"expose":false,"readOnly":false,"transport":"json"}
// @Tags tls
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body tlscert.Update true "Request"
// @Success 200 {object} tlscert.View
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/tls [put]
func (s *Server) updateTLS(w http.ResponseWriter, r *http.Request) {
	if s.certificates == nil {
		writeError(w, http.StatusConflict, "tls is disabled")
		return
	}

	var update tlscert.Update
	if err := readJSON(r, &update); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	err := s.certificates.Update(update)
	switch {
	case errors.Is(err, tlscert.ErrLocked):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, tlscert.ErrInvalid):
		writeError(w, http.StatusBadRequest, err.Error())
	case err != nil:
		writeError(w, http.StatusInternalServerError, err.Error())
	default:
		writeJSON(w, http.StatusOK, s.certificates.View())
	}
}
