package api

import (
	"context"
	"errors"
	"github.com/m-vinc/maco/pkg/auth"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/m-vinc/maco/pkg/config"
	"github.com/m-vinc/maco/pkg/engine"
)

func TestCredentialRevocation(t *testing.T) {
	paths, err := config.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	s := New(paths, []byte("secret"), nil, nil)
	token, err := issueAPITestToken(t, s)
	if err != nil {
		t.Fatal(err)
	}

	request := func() int { return mediaStatus(s, token) }
	if status := request(); status != http.StatusOK {
		t.Fatalf("initial request: %d", status)
	}

	if err := s.engine.SetPassword(context.Background(), "test", "changed-password"); err != nil {
		t.Fatal(err)
	}

	if status := request(); status != http.StatusUnauthorized {
		t.Fatalf("password change did not revoke: %d", status)
	}

	user, err := s.engine.Authenticate(context.Background(), "test", "changed-password")
	if err != nil {
		t.Fatal(err)
	}

	token, err = auth.IssueUserToken(s.secret, user, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.engine.DeleteUser(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}

	if status := request(); status != http.StatusUnauthorized {
		t.Fatalf("deletion did not revoke: %d", status)
	}
}

func mediaStatus(s *Server, token string) int {
	r := httptest.NewRequest(http.MethodGet, "/api/media", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	return w.Code
}

func TestTokenCheckKeepsSessionOnTransientErrors(t *testing.T) {
	paths, err := config.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	s := New(paths, []byte("secret"), nil, nil)
	token, err := issueAPITestToken(t, s)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.authenticateToken(context.Background(), "invalid"); !errors.Is(err, engine.ErrInvalidCredentials) {
		t.Fatalf("invalid token error = %v", err)
	}

	stopped, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.authenticateToken(stopped, token); err == nil || errors.Is(err, engine.ErrInvalidCredentials) {
		t.Fatalf("cancelled check error = %v, want a transient error", err)
	}
}
