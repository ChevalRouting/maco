package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/m-vinc/maco/pkg/config"
	"github.com/m-vinc/maco/pkg/db"
	"github.com/m-vinc/maco/pkg/metrics"
)

type metricsAPICase struct {
	path   string
	status int
}

func metricsRequest(s *Server, token, path string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	return w
}

func TestMetricsEndpoints(t *testing.T) {
	paths, err := config.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	s := New(paths, []byte("secret"), nil, nil)
	token, err := issueAPITestToken(t, s)
	if err != nil {
		t.Fatal(err)
	}

	if response := metricsRequest(s, "", "/api/metrics?scope=host&target=host"); response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated history: %d", response.Code)
	}

	cases := []metricsAPICase{
		{path: "/api/metrics?scope=bad&target=host", status: http.StatusBadRequest},
		{path: "/api/metrics?scope=host&target=other", status: http.StatusBadRequest},
		{path: "/api/metrics?scope=vm", status: http.StatusBadRequest},
		{path: "/api/metrics?scope=host&target=host&range=forever", status: http.StatusBadRequest},
		{path: "/api/metrics?scope=host&target=host&range=7d", status: http.StatusOK},
		{path: "/api/metrics/interfaces", status: http.StatusOK},
	}
	for _, test := range cases {
		if response := metricsRequest(s, token, test.path); response.Code != test.status {
			t.Fatalf("%s: %d %s", test.path, response.Code, response.Body.String())
		}
	}

	database, err := s.engine.OpenDB(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	samples := []db.MetricSample{{Scope: "host", Target: "host", Point: metrics.Point{TS: time.Now().Unix(), MemoryBytes: 42}}}
	if err := database.SaveMetrics(context.Background(), samples, 0); err != nil {
		t.Fatal(err)
	}

	response := metricsRequest(s, token, "/api/metrics?scope=host&target=host")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"cpu_percent":null`) || !strings.Contains(response.Body.String(), `"memory_bytes":42`) {
		t.Fatalf("history response: %d %s", response.Code, response.Body.String())
	}
}
