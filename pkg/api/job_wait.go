package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/m-vinc/maco/pkg/jobs"
)

const jobWaitTimeout = 2 * time.Hour

// @Summary waitJob
// @ID waitJob
// @Description Wait for one job ID until completion through a finite server-sent event stream. The done event returns the final job. Failed jobs are errors; connection loss does not cancel or prove failure of the underlying job. Inspect getJob or wait again with the same ID.
// @x-maco {"expose":true,"readOnly":true,"transport":"sse","stream":{"event":"done","stateField":"state","failureValue":"failed"}}
// @Tags jobs
// @Security BearerAuth
// @Produce text/event-stream
// @Param id path string true "id"
// @Success 200 {object} jobs.Job
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/jobs/{id}/wait [get]
func (s *Server) waitJob(w http.ResponseWriter, r *http.Request) {
	if s.jobs == nil {
		writeError(w, http.StatusServiceUnavailable, "job queue unavailable")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	id := chi.URLParam(r, "id")

	ctx, cancel := context.WithTimeout(r.Context(), jobWaitTimeout)
	defer cancel()

	subscription, err := s.jobs.Subscribe(ctx)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "job stream unavailable")
		return
	}
	defer func() { _ = subscription.Close() }()

	current, err := s.jobs.Get(ctx, id)
	if err != nil || current.Private() {
		writeError(w, http.StatusNotFound, "job unavailable")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	if terminalState(current.State) {
		_ = writeJobWaitEvent(w, flusher, "done", current)
		return
	}
	if err := writeJobWaitEvent(w, flusher, "update", current); err != nil {
		return
	}

	events := subscription.Channel()
	ping := time.NewTicker(20 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ping.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case msg, ok := <-events:
			if !ok {
				return
			}
			var job jobs.Job
			if err := json.Unmarshal([]byte(msg.Payload), &job); err != nil || job.ID != id {
				continue
			}
			job.Label = jobs.Label(job.Action)
			if terminalState(job.State) {
				if full, err := s.jobs.Get(ctx, id); err == nil {
					job = *full
				}
				_ = writeJobWaitEvent(w, flusher, "done", &job)
				return
			}
			if err := writeJobWaitEvent(w, flusher, "update", &job); err != nil {
				return
			}
		}
	}
}

func terminalState(state jobs.State) bool {
	return state == jobs.Succeeded || state == jobs.Failed
}

func writeJobWaitEvent(w http.ResponseWriter, flusher http.Flusher, event string, job *jobs.Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
