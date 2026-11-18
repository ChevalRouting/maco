package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/m-vinc/maco/pkg/jobs"
)

func (s *Server) submitJob(w http.ResponseWriter, r *http.Request, payload jobs.Payload) {
	if s.jobs == nil {
		writeError(w, http.StatusServiceUnavailable, "job queue unavailable")
		return
	}

	j, err := s.jobs.Submit(r.Context(), payload)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "could not enqueue job")
		return
	}

	writeJSON(w, http.StatusAccepted, j)
}

// @Summary listJobs
// @ID listJobs
// @Description List asynchronous jobs with state, IDs, and logs. Filter by state or search and use page/page_size for pagination. States are pending, running, succeeded, or failed.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags jobs
// @Security BearerAuth
// @Produce json
// @Param page query int false "page"
// @Param page_size query int false "page_size"
// @Param state query string false "state"
// @Param search query string false "search"
// @Param focus query string false "focus"
// @Success 200 {object} jobs.Page
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/jobs [get]
func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	options := jobs.PageOptions{Page: 1, PageSize: 25, State: r.URL.Query().Get("state"), Search: r.URL.Query().Get("search"), Focus: r.URL.Query().Get("focus")}
	var err error
	if value := r.URL.Query().Get("page"); value != "" {
		options.Page, err = strconv.Atoi(value)
		if err != nil || options.Page < 1 {
			writeError(w, http.StatusBadRequest, "invalid page")
			return
		}
	}

	if value := r.URL.Query().Get("page_size"); value != "" {
		options.PageSize, err = strconv.Atoi(value)
		if err != nil || options.PageSize < 1 || options.PageSize > 100 {
			writeError(w, http.StatusBadRequest, "page_size must be between 1 and 100")
			return
		}
	}

	switch options.State {
	case "", "all", "active", "completed", "pending", "running", "succeeded", "failed":
	default:
		writeError(w, http.StatusBadRequest, "invalid job state")
		return
	}

	list, err := s.jobs.PublicPage(r.Context(), options)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "job store unavailable")
		return
	}

	writeJSON(w, http.StatusOK, list)
}

// @Summary getJob
// @ID getJob
// @Description Read a job by its returned job ID, including state, logs, result, and error. succeeded confirms completion; pending or running does not. A failed job may already have produced partial effects; inspect it before repeating the action.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags jobs
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} jobs.Job
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/jobs/{id} [get]
func (s *Server) getJob(w http.ResponseWriter, r *http.Request) {
	j, err := s.jobs.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil || j.Private() {
		writeError(w, http.StatusNotFound, "job unavailable")
		return
	}

	writeJSON(w, http.StatusOK, j)
}
