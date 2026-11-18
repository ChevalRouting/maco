package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRoutesRequireAuth(t *testing.T) {
	s := &Server{}
	s.router = s.buildRouter()

	cases := []protectedRoute{
		{http.MethodPost, "/api/host/poweroff"},
		{http.MethodPost, "/api/host/reboot"},
		{http.MethodPut, "/api/tls"},
		{http.MethodGet, "/api/jobs/anything/wait"},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, httptest.NewRequest(c.method, c.route, nil))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without a token returned %d, want 401", c.method, c.route, w.Code)
		}
	}
}

type protectedRoute struct {
	method string
	route  string
}
