package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/yaml.v3"
)

type liveFixture struct {
	mu       sync.Mutex
	document map[string]any
	offline  bool
	name     string
}

func fixture(t *testing.T, name string) *liveFixture {
	t.Helper()
	f := &liveFixture{name: name}
	if err := json.Unmarshal(testContract, &f.document); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *liveFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.URL.Path == "/api/docs/openapi.json" {
		if f.offline {
			http.Error(w, "host unavailable", 503)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(f.document)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/wait") || strings.HasSuffix(r.URL.Path, "/completion") {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "event: done\ndata: {\"id\":\"job-9\",\"state\":\"succeeded\"}\n\n")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "POST" {
		w.WriteHeader(202)
		_, _ = fmt.Fprint(w, `{"id":"job-9","state":"pending"}`)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"host": f.name})
}

func serverForFixtures(t *testing.T, fixtures map[string]*liveFixture) (*mcp.ClientSession, *Server) {
	t.Helper()
	cfg := FileConfig{Instances: map[string]InstanceConfig{}}
	for name, f := range fixtures {
		host := httptest.NewServer(f)
		t.Cleanup(host.Close)
		cfg.Instances[name] = InstanceConfig{URL: host.URL, Token: "test-secret"}
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "registry.yml")
	if err = os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	server, err := New(file, "test")
	if err != nil {
		t.Fatal(err)
	}
	return connectServer(t, server), server
}

func TestMixedContractsDoNotShareInputSchemas(t *testing.T) {
	a, b := fixture(t, "a"), fixture(t, "b")
	paths := b.document["paths"].(map[string]any)
	op := paths["/api/vms"].(map[string]any)["get"].(map[string]any)
	op["parameters"] = []any{map[string]any{"name": "scope", "in": "query", "required": true, "schema": map[string]any{"type": "string"}}}
	cs, s := serverForFixtures(t, map[string]*liveFixture{"a": a, "b": b})
	if len(s.failures) > 0 {
		t.Fatal(s.failures)
	}
	toolA := s.catalogs["a"].toolNames["listVMs"]
	toolB := s.catalogs["b"].toolNames["listVMs"]
	if toolA == toolB {
		t.Fatal("incompatible schemas share a tool")
	}
	if call(t, cs, toolA, map[string]any{"instance": "a"}).IsError {
		t.Fatal("host a rejected its own schema")
	}
	if !call(t, cs, toolB, map[string]any{"instance": "b"}).IsError {
		t.Fatal("host b required parameter ignored")
	}
	if !call(t, cs, toolA, map[string]any{"instance": "b"}).IsError {
		t.Fatal("host b accepted host a contract")
	}
	if call(t, cs, toolB, map[string]any{"instance": "b", "scope": "all"}).IsError {
		t.Fatal("host b valid request failed")
	}
}

func TestOfflineInstanceAndRefresh(t *testing.T) {
	a, b := fixture(t, "a"), fixture(t, "b")
	b.offline = true
	cs, s := serverForFixtures(t, map[string]*liveFixture{"a": a, "b": b})
	if s.catalogs["a"] == nil || s.catalogs["b"] != nil {
		t.Fatal("unexpected discovery state")
	}
	if call(t, cs, "listVMs", map[string]any{"instance": "a"}).IsError {
		t.Fatal("offline host blocked healthy host")
	}
	b.mu.Lock()
	b.offline = false
	b.mu.Unlock()
	if call(t, cs, "instances_refresh", map[string]any{"instance": "b"}).IsError {
		t.Fatal("refresh failed")
	}
	if call(t, cs, "listVMs", map[string]any{"instance": "b"}).IsError {
		t.Fatal("recovered host unavailable")
	}
	b.mu.Lock()
	delete(b.document["paths"].(map[string]any)["/api/vms"].(map[string]any), "get")
	b.mu.Unlock()
	call(t, cs, "instances_refresh", map[string]any{"instance": "b"})
	if !call(t, cs, "listVMs", map[string]any{"instance": "b"}).IsError {
		t.Fatal("removed operation still available on refreshed host")
	}
}

func TestContractMetadataDrivesRenamedJobOperations(t *testing.T) {
	f := fixture(t, "renamed")
	paths := f.document["paths"].(map[string]any)
	for _, raw := range paths {
		for _, entry := range raw.(map[string]any) {
			op := entry.(map[string]any)
			if meta, ok := op["x-maco"].(map[string]any); ok {
				if wait, ok := meta["wait"].(map[string]any); ok {
					wait["operationId"] = "awaitCompletion"
				}
			}
		}
	}
	wait := paths["/api/jobs/{id}/wait"].(map[string]any)
	wait["get"].(map[string]any)["operationId"] = "awaitCompletion"
	delete(paths, "/api/jobs/{id}/wait")
	paths["/api/tasks/{id}/completion"] = wait
	paths["/api/vms/{id}/backups"].(map[string]any)["post"].(map[string]any)["operationId"] = "protectVM"
	cs, s := serverForFixtures(t, map[string]*liveFixture{"renamed": f})
	if len(s.failures) > 0 {
		t.Fatal(s.failures)
	}
	result := call(t, cs, "protectVM", map[string]any{"instance": "renamed", "id": "vm-1", "wait": true})
	if result.IsError || !strings.Contains(result.Content[0].(*mcp.TextContent).Text, `"state":"succeeded"`) {
		t.Fatal(result)
	}
}

func TestDescribeContainsUsableContract(t *testing.T) {
	cs, _ := serverForFixtures(t, map[string]*liveFixture{"test": fixture(t, "test")})
	result := call(t, cs, "instance_describe", map[string]any{"instance": "test", "operation": "createVM"})
	if result.IsError {
		t.Fatal(result)
	}
	data, _ := json.Marshal(result.StructuredContent)
	for _, word := range []string{"MiB", "GiB", "username", "minimum", "responses", "waitJob", "listCatalog"} {
		if !strings.Contains(string(data), word) {
			t.Errorf("contract missing %s", word)
		}
	}
}

func TestInvalidContractDoesNotFallBack(t *testing.T) {
	f := fixture(t, "old")
	delete(f.document, "x-maco")
	cs, s := serverForFixtures(t, map[string]*liveFixture{"old": f})
	if s.catalogs["old"] != nil {
		t.Fatal("old server silently accepted")
	}
	list, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Tools) != 3 {
		t.Fatal("stale embedded tools exposed")
	}
	result := call(t, cs, "instances_list", map[string]any{})
	if !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "upgrade Maco") {
		t.Fatal("missing upgrade guidance")
	}
}

func TestContractCannotRedirectAPIRequests(t *testing.T) {
	for _, route := range []string{"https://evil.invalid/api/vms", "//evil.invalid/api/vms", "/api/../outside", "/api/vms?token=x", "/api/%2e%2e/outside"} {
		t.Run(route, func(t *testing.T) {
			f := fixture(t, "unsafe")
			paths := f.document["paths"].(map[string]any)
			paths[route] = paths["/api/vms"]
			delete(paths, "/api/vms")
			_, s := serverForFixtures(t, map[string]*liveFixture{"unsafe": f})
			if s.catalogs["unsafe"] != nil {
				t.Fatal("unsafe route accepted")
			}
		})
	}
}
