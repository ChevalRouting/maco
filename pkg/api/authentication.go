package api

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/m-vinc/maco/pkg/auth"
	"github.com/m-vinc/maco/pkg/engine"
	"golang.org/x/time/rate"
)

type loginGuard struct {
	mu      sync.Mutex
	clients map[string]*loginClient
	global  *rate.Limiter
	slots   chan struct{}
}

type loginClient struct {
	limiter *rate.Limiter
	seen    time.Time
}

func (g *loginGuard) acquire(r *http.Request) (func(), bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.global == nil {
		g.global = rate.NewLimiter(2, 8)
		g.clients = make(map[string]*loginClient)
		g.slots = make(chan struct{}, 4)
	}

	now := time.Now()
	for ip, client := range g.clients {
		if now.Sub(client.seen) > 10*time.Minute {
			delete(g.clients, ip)
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}

	client := g.clients[ip]
	if client == nil {
		if len(g.clients) >= 4096 {
			return nil, false
		}

		client = &loginClient{limiter: rate.NewLimiter(rate.Every(12*time.Second), 5)}
		g.clients[ip] = client
	}

	client.seen = now
	if !g.global.Allow() || !client.limiter.Allow() {
		return nil, false
	}

	select {
	case g.slots <- struct{}{}:
		return func() { <-g.slots }, true
	default:
		return nil, false
	}
}

func (s *Server) authenticate(ctx context.Context, token string) (*auth.Claims, error) {
	if strings.HasPrefix(token, engine.APIKeyPrefix) {
		return s.authenticateAPIKey(ctx, token)
	}

	return s.authenticateToken(ctx, token)
}

func (s *Server) authenticateAPIKey(ctx context.Context, token string) (*auth.Claims, error) {
	user, err := s.engine.AuthenticateAPIKey(ctx, token)
	if err != nil {
		return nil, err
	}

	return &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: user.Username},
		UserID:           user.ID,
		Role:             user.Role,
	}, nil
}

func (s *Server) authenticateToken(ctx context.Context, token string) (*auth.Claims, error) {
	claims, err := auth.ParseClaims(s.secret, token)
	if err != nil {
		return nil, err
	}

	if claims.UserID == "" || claims.Credentials == "" {
		return nil, fmt.Errorf("session must be renewed")
	}

	database, err := s.engine.OpenDB(ctx)
	if err != nil {
		return nil, err
	}

	user, found, err := database.GetUserByUsername(ctx, claims.Subject)
	if err != nil {
		return nil, err
	}

	if !found || user.ID != claims.UserID || subtle.ConstantTimeCompare([]byte(claims.Credentials), []byte(auth.CredentialVersion(user))) != 1 {
		return nil, fmt.Errorf("session revoked")
	}

	claims.Role = user.Role
	return claims, nil
}

type contextKey int

const claimsContextKey contextKey = iota

func withClaims(ctx context.Context, claims *auth.Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

func claimsFrom(ctx context.Context) *auth.Claims {
	claims, _ := ctx.Value(claimsContextKey).(*auth.Claims)
	return claims
}

func readOnlyMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

type CurrentUser struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

// @Summary currentUser
// @ID currentUser
// @Description Read the authenticated account name and role. Viewer accounts can inspect resources; administrator access is required for VM and host mutations.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags login
// @Security BearerAuth
// @Produce json
// @Success 200 {object} CurrentUser
// @Failure 401 {object} ErrorResponse
// @Router /api/me [get]
func (s *Server) currentUser(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, CurrentUser{Username: claims.Subject, Role: claims.Role})
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// @Summary changePassword
// @ID changePassword
// @Description Change the current account password by supplying current_password and new_password. Existing login sessions may become invalid. The response contains a new login token; treat it as a secret.
// @x-maco {"expose":true,"readOnly":false,"transport":"json"}
// @Tags login
// @Security BearerAuth
// @Produce json
// @Accept json
// @Param body body ChangePasswordRequest true "Request"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/me/password [patch]
func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req ChangePasswordRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "new password must be at least 8 characters")
		return
	}

	if _, err := s.engine.Authenticate(r.Context(), claims.Subject, req.CurrentPassword); err != nil {
		writeError(w, http.StatusUnauthorized, "current password is incorrect")
		return
	}

	if err := s.engine.SetPassword(r.Context(), claims.Subject, req.NewPassword); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	user, err := s.engine.Authenticate(r.Context(), claims.Subject, req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	token, err := auth.IssueUserToken(s.secret, user, 24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{Token: token})
}
