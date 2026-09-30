package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/m-vinc/maco/pkg/auth"
	"github.com/m-vinc/maco/pkg/jobs"
)

type JobsEvent struct {
	Type      string     `json:"type"`
	Resources []string   `json:"resources,omitempty" binding:"optional"`
	Jobs      []jobs.Job `json:"jobs,omitempty" binding:"optional"`
	Job       *jobs.Job  `json:"job,omitempty" binding:"optional"`
}

func (s *Server) acceptAuthenticated(w http.ResponseWriter, r *http.Request) (*websocket.Conn, *auth.Claims, func(), error) {
	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	ws.SetReadLimit(4096)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var credentials ConsoleAuth
	if err := wsjson.Read(ctx, ws, &credentials); err != nil {
		_ = ws.Close(websocket.StatusPolicyViolation, "authentication required")
		_ = ws.CloseNow()
		return nil, nil, nil, err
	}

	claims, err := s.authenticateToken(ctx, credentials.Token)
	if err != nil {
		_ = ws.Close(websocket.StatusPolicyViolation, "unauthorized")
		_ = ws.CloseNow()
		return nil, nil, nil, err
	}

	sessionCtx, stop := context.WithCancel(r.Context())
	go func() {
		expiry := time.NewTimer(time.Until(claims.ExpiresAt.Time))
		defer expiry.Stop()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-sessionCtx.Done():
				return
			case <-expiry.C:
				_ = ws.Close(websocket.StatusPolicyViolation, "session expired")
				return
			case <-ticker.C:
				checkCtx, cancel := context.WithTimeout(sessionCtx, 3*time.Second)
				_, err := s.authenticateToken(checkCtx, credentials.Token)
				cancel()
				if err != nil {
					_ = ws.Close(websocket.StatusPolicyViolation, "session revoked")
					return
				}
			}
		}
	}()
	return ws, claims, stop, nil
}

// @Summary streamJobs
// @ID streamJobs
// @Description Continuous job notifications over WebSocket. Use listJobs, getJob, or the finite waitJob operation from MCP.
// @x-maco {"expose":false,"readOnly":true,"transport":"unsupported"}
// @Tags jobs
// @Success 101 "WebSocket upgrade"
// @Router /api/jobs/stream [get]
func (s *Server) streamJobs(w http.ResponseWriter, r *http.Request) { s.stream(w, r, false) }

// @Summary streamEvents
// @ID streamEvents
// @Description Continuous resource notifications over WebSocket. Use read-only list and inspect operations from MCP.
// @x-maco {"expose":false,"readOnly":true,"transport":"unsupported"}
// @Tags events
// @Success 101 "WebSocket upgrade"
// @Router /api/events [get]
func (s *Server) streamEvents(w http.ResponseWriter, r *http.Request) { s.stream(w, r, true) }
func (s *Server) stream(w http.ResponseWriter, r *http.Request, generalized bool) {
	if s.jobs == nil {
		writeError(w, http.StatusServiceUnavailable, "job queue unavailable")
		return
	}

	ws, _, release, err := s.acceptAuthenticated(w, r)
	if err != nil {
		return
	}
	defer release()
	defer func() { _ = ws.CloseNow() }()
	ctx := ws.CloseRead(r.Context())

	subscribeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	subscription, err := s.jobs.Subscribe(subscribeCtx)
	cancel()
	if err != nil {
		_ = ws.Close(websocket.StatusInternalError, "job stream unavailable")
		return
	}
	defer func() { _ = subscription.Close() }()

	var resources <-chan []string
	if generalized {
		channel, unsubscribe := s.subscribeResources()
		defer unsubscribe()
		resources = channel
	}
	events := subscription.Channel()
	if err := s.sendJobsSnapshot(ctx, ws); err != nil {
		return
	}

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case changed := <-resources:
			if err := writeJobsEvent(ctx, ws, JobsEvent{Type: "invalidate", Resources: changed}); err != nil {
				return
			}
		case event, ok := <-events:
			if !ok {
				return
			}

			if err := s.sendStreamUpdate(ctx, ws, event.Payload, generalized); err != nil {
				_ = ws.Close(websocket.StatusInternalError, "job stream unavailable")
				return
			}
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := ws.Ping(pingCtx)
			cancel()
			if err != nil {
				_ = ws.Close(websocket.StatusInternalError, "job stream unavailable")
				return
			}
		}
	}
}

func (s *Server) sendJobsSnapshot(ctx context.Context, ws *websocket.Conn) error {
	readCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	list, err := s.jobs.PublicList(readCtx)
	if err != nil {
		return err
	}

	return writeJobsEvent(ctx, ws, JobsEvent{Type: "snapshot", Jobs: list})
}

func (s *Server) sendStreamUpdate(ctx context.Context, ws *websocket.Conn, payload string, generalized bool) error {
	var job jobs.Job
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		return err
	}
	job.Label = jobs.Label(job.Action)

	if job.Private() {
		if generalized && job.State == jobs.Succeeded {
			return writeJobsEvent(ctx, ws, JobsEvent{Type: "invalidate", Resources: []string{"preview:" + job.Target}})
		}
		return nil
	}

	if err := writeJobsEvent(ctx, ws, JobsEvent{Type: "update", Job: &job}); err != nil {
		return err
	}
	if generalized && (job.State == jobs.Succeeded || job.State == jobs.Failed) {
		resources := []string{"vms", "disks", "storage", "usb"}
		if strings.HasPrefix(job.Action, "network.") || strings.HasPrefix(job.Action, "vm.interface.") {
			resources = []string{"networks", "interfaces", "vms"}
		}
		return writeJobsEvent(ctx, ws, JobsEvent{Type: "invalidate", Resources: resources})
	}
	return nil
}

func writeJobsEvent(ctx context.Context, ws *websocket.Conn, event JobsEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return ws.Write(writeCtx, websocket.MessageText, data)
}
