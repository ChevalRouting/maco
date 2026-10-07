package api

import (
	"context"
	"github.com/m-vinc/maco/pkg/auth"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/m-vinc/maco/pkg/config"
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

	request := func() int {
		r := httptest.NewRequest(http.MethodGet, "/api/media", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		return w.Code
	}
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
