package api

import (
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/m-vinc/maco/pkg/image"
	"github.com/m-vinc/maco/pkg/jobs"
)

// @Summary listImages
// @ID listImages
// @Description List images currently available on this Maco host. Compare with listCatalog to discover downloadable images.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags images
// @Security BearerAuth
// @Produce json
// @Success 200 {array} string
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/images [get]
func (s *Server) listImages(w http.ResponseWriter, r *http.Request) {
	names := make([]string, 0, len(image.Catalog))
	for name := range image.Catalog {
		names = append(names, name)
	}

	media, err := s.engine.ListMedia()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	for _, m := range media {
		if m.Kind == "image" {
			names = append(names, "media:"+m.ID)
		}
	}
	sort.Strings(names)
	writeJSON(w, http.StatusOK, names)
}

// @Summary listCatalog
// @ID listCatalog
// @Description List supported image catalog entries, download availability, and guest provisioning capabilities. Use returned catalog keys instead of guessing image names.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags catalog
// @Security BearerAuth
// @Produce json
// @Success 200 {array} engine.CatalogImage
// @Failure 401 {object} ErrorResponse
// @Router /api/catalog [get]
func (s *Server) listCatalog(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.engine.Catalog())
}

// @Summary downloadCatalogImage
// @ID downloadCatalogImage
// @Description Download the selected catalog image to this host. Use its ID from listCatalog; wait for the download job before relying on the image being available. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags catalog
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Success 202 {object} jobs.Job
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/catalog/{id}/download [post]
func (s *Server) downloadCatalogImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s.submitJob(w, r, jobs.Payload{Action: "catalog.download", Target: id})
}

// @Summary deleteCatalogImage
// @ID deleteCatalogImage
// @Description Delete the downloaded copy of a catalog image from this host. Use the catalog ID from listCatalog. Existing VM dependencies may prevent removal. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json"}
// @Tags catalog
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Success 204 "No content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/catalog/{id} [delete]
func (s *Server) deleteCatalogImage(w http.ResponseWriter, r *http.Request) {
	if err := s.engine.DeleteCatalogImage(chi.URLParam(r, "id")); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
