package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/m-vinc/maco/pkg/engine"
	"github.com/rs/zerolog/log"
)

type CreateMediaRequest struct {
	Name    string `json:"name"`
	SizeGiB int    `json:"size_gib"`
}

// @Summary listMedia
// @ID listMedia
// @Description List uploaded ISO files and disk images with their IDs and usage. Use returned media IDs in VM creation and updateMedia requests.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags media
// @Security BearerAuth
// @Produce json
// @Success 200 {array} engine.Media
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/media [get]
func (s *Server) listMedia(w http.ResponseWriter, r *http.Request) {
	m, err := s.engine.ListMedia()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	writeJSON(w, 200, m)
}

// @Summary createMedia
// @ID createMedia
// @Description Create a blank named disk image in the media library. size_gib is the requested capacity in GiB; consult diskStorage first. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json"}
// @Tags media
// @Security BearerAuth
// @Produce json
// @Accept json
// @Param body body CreateMediaRequest true "Request"
// @Success 201 {object} engine.Media
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/media [post]
func (s *Server) createMedia(w http.ResponseWriter, r *http.Request) {
	var p CreateMediaRequest
	if err := readJSON(r, &p); err != nil {
		writeError(w, 400, err.Error())
		return
	}

	m, err := s.engine.CreateMedia(r.Context(), p.Name, p.SizeGiB, nil)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}

	writeJSON(w, 201, m)
}

const maxISOUpload = 10 << 30

// @Summary uploadISO
// @ID uploadISO
// @Description Upload an ISO file from the MCP host into Maco media storage, up to 10 GiB including multipart framing. Use an absolute local file_path. Returns the new media record; it does not attach the ISO to a VM. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"multipart","upload":{"field":"file","maxBytes":10737418240,"extensions":[".iso"]}}
// @Tags media
// @Security BearerAuth
// @Produce json
// @Accept multipart/form-data
// @Param file formData file true "ISO file, maximum 10 GiB"
// @Success 201 {object} engine.Media
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 413 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 507 {object} ErrorResponse
// @Router /api/media/iso [post]
func (s *Server) uploadISO(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength > maxISOUpload {
		writeError(w, 413, "ISO exceeds the 10 GiB upload limit")
		return
	}

	if r.ContentLength > 0 {
		free, err := s.engine.MediaFreeBytes()
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}

		if uint64(r.ContentLength) > free {
			writeError(w, 507, "not enough free disk space for this ISO")
			return
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxISOUpload)
	reader, err := r.MultipartReader()
	if err != nil {
		writeError(w, 400, "expected multipart ISO upload")
		return
	}

	part, err := reader.NextPart()
	log.Info().Msgf("%+v %+v", part, err)
	if err != nil || part.FormName() != "file" || !strings.HasSuffix(strings.ToLower(part.FileName()), ".iso") {
		log.Info().Msgf("%+v", part)
		writeError(w, 400, "select an ISO file")
		return
	}

	defer func() { _ = part.Close() }()
	m, err := s.engine.CreateMedia(r.Context(), part.FileName(), 0, part)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}

	writeJSON(w, 201, m)
}

const maxImageUpload = 64 << 30

// @Summary uploadImage
// @ID uploadImage
// @Description Upload a .qcow2 or .img disk image from the MCP host, up to 64 GiB including multipart framing. Use an absolute local file_path. Inspect the returned media record before attaching or using it. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"multipart","upload":{"field":"file","maxBytes":68719476736,"extensions":[".qcow2",".img"]}}
// @Tags media
// @Security BearerAuth
// @Produce json
// @Accept multipart/form-data
// @Param file formData file true "Disk image (.qcow2 or .img), maximum 64 GiB"
// @Success 201 {object} engine.Media
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 413 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 507 {object} ErrorResponse
// @Router /api/media/image [post]
func (s *Server) uploadImage(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength > maxImageUpload {
		writeError(w, 413, "image exceeds the 64 GiB upload limit")
		return
	}

	if r.ContentLength > 0 {
		free, err := s.engine.MediaFreeBytes()
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}

		if uint64(r.ContentLength) > free {
			writeError(w, 507, "not enough free disk space for this image")
			return
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImageUpload)
	reader, err := r.MultipartReader()
	if err != nil {
		writeError(w, 400, "expected multipart image upload")
		return
	}

	part, err := reader.NextPart()
	if err != nil || part.FormName() != "file" || !isDiskImage(part.FileName()) {
		writeError(w, 400, "select a .qcow2 or .img disk image")
		return
	}

	defer func() { _ = part.Close() }()
	m, err := s.engine.UploadImage(r.Context(), part.FileName(), part)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}

	writeJSON(w, 201, m)
}

func isDiskImage(name string) bool {
	name = strings.ToLower(name)
	return strings.HasSuffix(name, ".qcow2") || strings.HasSuffix(name, ".img")
}

// @Summary deleteMedia
// @ID deleteMedia
// @Description Delete the selected media library entry by ID from listMedia. This is destructive and can be rejected if media is still in use. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json"}
// @Tags media
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Success 204 "No content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/media/{id} [delete]
func (s *Server) deleteMedia(w http.ResponseWriter, r *http.Request) {
	if err := s.engine.DeleteMedia(chi.URLParam(r, "id")); err != nil {
		writeError(w, 400, err.Error())
		return
	}

	w.WriteHeader(204)
}

// @Summary updateMedia
// @ID updateMedia
// @Description Update a VM ISO attachment and boot-order settings. Discover media IDs with listMedia and current settings with getVM. Boot entries use disk, disk:<disk-id>, and iso:<media-id>. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json"}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Accept json
// @Param body body engine.MediaSettings true "Request"
// @Success 200 {object} engine.MediaSettings
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/vms/{id}/media [patch]
func (s *Server) updateMedia(w http.ResponseWriter, r *http.Request) {
	var p engine.MediaSettings
	if err := readJSON(r, &p); err != nil {
		writeError(w, 400, err.Error())
		return
	}

	if err := s.engine.UpdateMediaContext(r.Context(), chi.URLParam(r, "id"), p); err != nil {
		writeError(w, 400, err.Error())
		return
	}

	writeJSON(w, 200, p)
}
