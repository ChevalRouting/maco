package api

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/m-vinc/maco/pkg/auth"
	"github.com/m-vinc/maco/pkg/config"
	"github.com/m-vinc/maco/pkg/engine"
	"github.com/m-vinc/maco/pkg/jobs"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Server struct {
	events resourceHub
	login  loginGuard
	engine *engine.Engine
	jobs   *jobs.Service
	secret []byte
	static fs.FS
	router chi.Router
}

func New(paths *config.Paths, secret []byte, static fs.FS, queue *jobs.Service) *Server {
	s := &Server{engine: engine.New(paths), secret: secret, static: static, jobs: queue}
	s.router = s.buildRouter()
	return s
}

func (s *Server) Handler() http.Handler { return s.router }

func (s *Server) buildRouter() chi.Router {
	r := chi.NewRouter()

	r.Use(requestLogger)
	r.Use(recoverPanics)
	r.Use(securityHeaders)
	r.Route("/api", s.mountAPI)

	s.mountStatic(r)
	return r
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Referrer-Policy", "no-referrer")
		header.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; font-src 'self' data:; connect-src 'self' ws: wss:; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) mountAPI(r chi.Router) {
	r.Get("/docs/openapi.json", s.serveOpenAPI)
	r.Post("/login", s.handleLogin)
	r.Get("/vms/{id}/console", s.consoleVM)
	r.Get("/vms/{id}/display", s.displayVM)
	r.Get("/host/console", s.hostConsole)
	r.Get("/jobs/stream", s.streamJobs)
	r.Get("/events", s.streamEvents)
	r.Group(s.mountProtected)
}

func (s *Server) mountProtected(r chi.Router) {
	r.Use(s.requireAuth)
	r.Use(s.notifyMutations)
	r.Get("/me", s.currentUser)
	r.Patch("/me/password", s.changePassword)
	r.Get("/me/api-keys", s.listAPIKeys)
	r.Post("/me/api-keys", s.createAPIKey)
	r.Delete("/me/api-keys/{id}", s.revokeAPIKey)
	r.Get("/me/ssh-keys", s.listSSHKeys)
	r.Post("/me/ssh-keys", s.addSSHKey)
	r.Delete("/me/ssh-keys/{id}", s.deleteSSHKey)
	r.Get("/usb/devices", s.listUSBDevices)
	r.Get("/vms/{id}/usb", s.listVMUSB)
	r.Post("/vms/{id}/usb", s.attachUSB)
	r.Post("/vms/{id}/usb/assignments", s.assignUSB)
	r.Delete("/vms/{id}/usb/assignments/{key}", s.unassignUSB)
	r.Delete("/vms/{id}/usb/{attachment}", s.detachUSB)
	r.Get("/vms", s.listVMs)
	r.Post("/vms", s.createVM)
	r.Get("/vms/{id}", s.getVM)
	r.Patch("/vms/{id}/hardware", s.updateHardware)
	r.Get("/interfaces/mac", s.suggestMAC)
	r.Get("/vms/{id}/guest-setup", s.getGuestSetup)
	r.Patch("/vms/{id}/guest-setup", s.updateGuestSetup)
	r.Post("/vms/{id}/interfaces", s.addVMInterface)
	r.Patch("/vms/{id}/interfaces/{interface}", s.updateVMInterface)
	r.Delete("/vms/{id}/interfaces/{interface}", s.removeVMInterface)
	r.Post("/vms/{id}/disks", s.addDisk)
	r.Patch("/vms/{id}/disks/{disk}", s.growDisk)
	r.Delete("/vms/{id}/disks/{disk}", s.removeDisk)
	r.Post("/vms/{id}/disks/{disk}/wipe", s.wipeDisk)
	r.Post("/vms/{id}/disks/{disk}/replace", s.replaceDisk)
	r.Get("/vms/{id}/preview", s.previewVM)
	r.Get("/vms/{id}/guest-agent", s.vmGuestAgent)
	r.Post("/vms/{id}/start", s.startVM)
	r.Post("/vms/{id}/stop", s.stopVM)
	r.Post("/vms/{id}/shutdown", s.shutdownVM)
	r.Post("/vms/{id}/reboot", s.rebootVM)
	r.Get("/vms/{id}/backups", s.listBackups)
	r.Post("/vms/{id}/backups", s.createBackup)
	r.Get("/vms/{id}/backups/schedule", s.getBackupSchedule)
	r.Put("/vms/{id}/backups/schedule", s.setBackupSchedule)
	r.Post("/vms/{id}/backups/{timestamp}/restore", s.restoreBackup)
	r.Delete("/vms/{id}/backups/{timestamp}", s.deleteBackup)
	r.Get("/vms/{id}/snapshots", s.listSnapshots)
	r.Post("/vms/{id}/snapshots", s.createSnapshot)
	r.Post("/vms/{id}/snapshots/{tag}/restore", s.restoreSnapshot)
	r.Delete("/vms/{id}/snapshots/{tag}", s.deleteSnapshot)
	r.Delete("/vms/{id}", s.deleteVM)
	r.Get("/networks", s.listNetworks)
	r.Post("/networks", s.createNetwork)
	r.Put("/networks/{id}", s.updateNetwork)
	r.Post("/networks/{id}/apply", s.applyNetwork)
	r.Delete("/networks/{id}", s.destroyNetwork)
	r.Get("/disks", s.listDisks)
	r.Get("/disks/stats", s.diskStorage)
	r.Get("/host", s.hostInfo)
	r.Post("/host/poweroff", s.powerOffHost)
	r.Post("/host/reboot", s.rebootHost)
	r.Get("/interfaces", s.listInterfaces)
	r.Get("/images", s.listImages)
	r.Get("/catalog", s.listCatalog)
	r.Post("/catalog/{id}/download", s.downloadCatalogImage)
	r.Delete("/catalog/{id}", s.deleteCatalogImage)
	r.Get("/media", s.listMedia)
	r.Post("/media", s.createMedia)
	r.Post("/media/iso", s.uploadISO)
	r.Post("/media/image", s.uploadImage)
	r.Delete("/media/{id}", s.deleteMedia)
	r.Patch("/vms/{id}/media", s.updateMedia)
	r.Get("/templates", s.listTemplates)
	r.Post("/templates", s.createTemplate)
	r.Get("/templates/{id}", s.getTemplate)
	r.Put("/templates/{id}", s.updateTemplate)
	r.Post("/templates/{id}/instantiate", s.instantiateTemplate)
	r.Delete("/templates/{id}", s.deleteTemplate)
	r.Get("/jobs", s.listJobs)
	r.Get("/jobs/{id}", s.getJob)
	r.Get("/jobs/{id}/wait", s.waitJob)
}

func (s *Server) mountStatic(r chi.Router) {
	if s.static != nil {
		r.NotFound(s.serveStatic)
	}
}

func (s *Server) serveStatic(w http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, "/api/") {
		writeError(w, http.StatusNotFound, "API route not found")
		return
	}

	path := strings.TrimPrefix(req.URL.Path, "/")
	if path != "" {
		if _, err := fs.Stat(s.static, path); err == nil {
			http.FileServer(http.FS(s.static)).ServeHTTP(w, req)
			return
		}
	}

	index, err := s.static.Open("index.html")
	if err != nil {
		http.NotFound(w, req)
		return
	}

	defer func() { _ = index.Close() }()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.Copy(w, index)
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		token := strings.TrimPrefix(header, "Bearer ")
		if token == header || token == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		claims, err := s.authenticate(r.Context(), token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		if !readOnlyMethod(r.Method) && !auth.CanMutate(claims.Role) && !strings.HasPrefix(r.URL.Path, "/api/me/") {
			writeError(w, http.StatusForbidden, "administrator role required")
			return
		}

		next.ServeHTTP(w, r.WithContext(withClaims(r.Context(), claims)))
	})
}

// @Summary handleLogin
// @ID handleLogin
// @Description Exchange an account username and password for a login token. MCP uses registry credentials and does not expose login as a tool.
// @x-maco {"expose":false,"readOnly":false,"transport":"unsupported"}
// @Tags login
// @Produce json
// @Accept json
// @Param body body LoginRequest true "Request"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/login [post]
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	release, allowed := s.login.acquire(r)
	if !allowed {
		w.Header().Set("Retry-After", "12")
		writeError(w, http.StatusTooManyRequests, "too many login attempts")
		return
	}

	defer release()
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(5 * time.Second))
	var req LoginRequest

	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	user, err := s.engine.Authenticate(r.Context(), req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := auth.IssueUserToken(s.secret, user, 24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{Token: token})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

func readJSON(r *http.Request, v any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("request must contain one JSON object")
	}

	return nil
}

func (s *Server) notifyMutations(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if strings.HasPrefix(r.URL.Path, "/api/media") {
				s.events.broadcast("media", "disks", "storage", "vms")
			}

			if strings.HasPrefix(r.URL.Path, "/api/templates") {
				s.events.broadcast("templates")
			}
		}
	})
}
