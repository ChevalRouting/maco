package mcpserver

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed testdata/openapi.json
var testContract []byte

func connectTest(t *testing.T, handler http.HandlerFunc) (*mcp.ClientSession, *Server) {
	t.Helper()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/docs/openapi.json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(testContract)
			return
		}

		handler(w, r)
	}))
	t.Cleanup(upstream.Close)
	config := filepath.Join(t.TempDir(), "instances.yml")
	if err := os.WriteFile(config, []byte(fmt.Sprintf("instances:\n  test:\n    url: %s\n    token: secret-test-token\n", upstream.URL)), 0600); err != nil {
		t.Fatal(err)
	}

	server, err := New(config, "test")
	if err != nil {
		t.Fatal(err)
	}

	if len(server.failures) > 0 {
		t.Fatal(server.failures)
	}

	return connectServer(t, server), server
}

func connectServer(t *testing.T, server *Server) *mcp.ClientSession {
	t.Helper()
	a, b := mcp.NewInMemoryTransports()
	session, err := server.mcp.Connect(context.Background(), a, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = session.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	cs, err := client.Connect(context.Background(), b, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func call(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}

	return result
}

func TestToolsCoverAPI(t *testing.T) {
	cs, _ := connectTest(t, func(http.ResponseWriter, *http.Request) { t.Error("unexpected request") })
	listed, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	names := map[string]*mcp.Tool{}
	for _, tool := range listed.Tools {
		names[tool.Name] = tool
	}

	var contract struct {
		Paths map[string]map[string]operation `json:"paths"`
	}

	if err := json.Unmarshal(testContract, &contract); err != nil {
		t.Fatal(err)
	}

	count := 3
	for _, methods := range contract.Paths {
		for _, op := range methods {
			if !op.Metadata.Expose {
				continue
			}

			count++
			tool, ok := names[op.ID]
			if !ok {
				t.Errorf("missing %s", op.ID)
				continue
			}

			if tool.Annotations.ReadOnlyHint != op.Metadata.ReadOnly {
				t.Errorf("incorrect read-only hint for %s", op.ID)
			}
		}
	}

	if len(names) != count {
		t.Errorf("got %d tools, want %d", len(names), count)
	}

	result := call(t, cs, "instances_list", map[string]any{})
	data, _ := json.Marshal(result)
	if strings.Contains(string(data), "secret-test-token") || !strings.Contains(string(data), "test") {
		t.Fatal(string(data))
	}
}

func TestRequests(t *testing.T) {
	var requests int
	cs, _ := connectTest(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") != "Bearer secret-test-token" {
			t.Error("missing token")
		}

		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /api/vms":
			_, _ = fmt.Fprint(w, `[{"phase":"running"}]`)
		case "GET /api/vms/vm one":
			_, _ = fmt.Fprint(w, `{"phase":"stopped"}`)
		case "POST /api/vms":
			data, _ := io.ReadAll(r.Body)
			if string(data) != `{"cpus":2,"disk_size_gib":20,"memory_mib":2048,"name":"new-vm","username":"maco"}` {
				t.Errorf("body %s", data)
			}

			if r.Header.Get("Content-Type") != "application/json" {
				t.Error("missing content type")
			}

			w.WriteHeader(202)
			_, _ = fmt.Fprint(w, `{"id":"job-1","state":"pending"}`)
		case "POST /api/vms/vm-1/backups":
			w.WriteHeader(202)
			_, _ = fmt.Fprint(w, `{"id":"backup-job"}`)
		case "GET /api/jobs":
			if r.URL.Query().Get("search") != "a&b" || r.URL.Query().Get("page") != "2" {
				t.Error(r.URL.RawQuery)
			}

			_, _ = fmt.Fprint(w, `{"items":[]}`)
		case "DELETE /api/vms/vm-1/backups/2026-09-26":
			w.WriteHeader(204)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			w.WriteHeader(404)
		}
	})
	for _, tc := range []struct {
		name string
		args map[string]any
	}{
		{"listVMs", map[string]any{}},
		{"getVM", map[string]any{"id": "vm one"}},
		{"createVM", map[string]any{"body": map[string]any{"name": "new-vm", "cpus": 2, "memory_mib": 2048, "disk_size_gib": 20, "username": "maco"}}},
		{"createBackup", map[string]any{"id": "vm-1"}},
		{"listJobs", map[string]any{"page": 2, "search": "a&b"}},
		{"deleteBackup", map[string]any{"id": "vm-1", "timestamp": "2026-09-26"}},
	} {
		tc.args["instance"] = "test"
		r := call(t, cs, tc.name, tc.args)
		if r.IsError {
			t.Fatalf("%s: %+v", tc.name, r)
		}
	}

	if requests != 6 {
		t.Fatalf("requests %d", requests)
	}

	for _, tc := range []struct {
		name string
		args map[string]any
	}{
		{"getVM", map[string]any{"instance": "test"}},
		{"getVM", map[string]any{"instance": "test", "id": "../host"}},
		{"getVM", map[string]any{"instance": "test", "id": ".."}},
		{"listVMs", map[string]any{"instance": "missing"}},
		{"createVM", map[string]any{"instance": "test", "body": map[string]any{"cpus": "two"}}},
		{"listVMs", map[string]any{"instance": "test", "url": "https://evil.invalid"}},
	} {
		if !call(t, cs, tc.name, tc.args).IsError {
			t.Errorf("%s: expected error", tc.name)
		}
	}

	if requests != 6 {
		t.Fatal("invalid inputs reached upstream")
	}
}

func TestSpecialResponses(t *testing.T) {
	cs, _ := connectTest(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/vms/vm/preview":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("png-test"))
		case "/api/jobs/job/wait":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprint(w, "event: update\ndata: {\"state\":\"active\"}\n\nevent: done\ndata: {\"state\":\"succeeded\"}\n\n")
		default:
			w.WriteHeader(403)
			_, _ = fmt.Fprint(w, `{"error":"administrator role required"}`)
		}
	})
	preview := call(t, cs, "previewVM", map[string]any{"instance": "test", "id": "vm"})
	img, ok := preview.Content[0].(*mcp.ImageContent)
	if !ok || string(img.Data) != "png-test" {
		t.Fatalf("preview %+v", preview)
	}

	waited := call(t, cs, "waitJob", map[string]any{"instance": "test", "id": "job"})
	if waited.IsError {
		t.Fatal(waited)
	}

	denied := call(t, cs, "powerOffHost", map[string]any{"instance": "test", "body": map[string]any{}})
	if !denied.IsError {
		t.Fatal("expected HTTP error")
	}
}

func TestUpload(t *testing.T) {
	cs, _ := connectTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/media/iso" {
			t.Error(r.URL.Path)
		}

		reader, err := r.MultipartReader()
		if err != nil {
			t.Error(err)
			return
		}

		part, err := reader.NextPart()
		if err != nil {
			t.Error(err)
			return
		}

		data, _ := io.ReadAll(part)
		if part.FormName() != "file" || part.FileName() != "test.iso" || string(data) != "iso-content" {
			t.Error("incorrect multipart upload")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id":"media-1"}`)
	})
	file := filepath.Join(t.TempDir(), "test.iso")
	if err := os.WriteFile(file, []byte("iso-content"), 0600); err != nil {
		t.Fatal(err)
	}

	result := call(t, cs, "uploadISO", map[string]any{"instance": "test", "file_path": file})
	if result.IsError {
		t.Fatal(result)
	}
}

func TestRedirectDoesNotForwardToken(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("followed redirect") }))
	defer target.Close()
	cs, _ := connectTest(t, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) })
	if !call(t, cs, "listVMs", map[string]any{"instance": "test"}).IsError {
		t.Fatal("redirect accepted")
	}
}

func TestJobStreamFailure(t *testing.T) {
	result, err := readJobStream(strings.NewReader("event: done\ndata: {\"state\":\"failed\"}\n\n"), streamMetadata{Event: "done", StateField: "state", FailureValue: "failed"})
	if err != nil || !result.IsError {
		t.Fatalf("%+v %v", result, err)
	}

	if _, err = readJobStream(strings.NewReader("event: update\ndata: {}\n\n"), streamMetadata{Event: "done", StateField: "state", FailureValue: "failed"}); err == nil {
		t.Fatal("incomplete stream accepted")
	}
}

func TestRegistryValidation(t *testing.T) {
	t.Setenv("MACO_TEST_TOKEN", "injected")
	for _, tc := range []struct {
		config string
		valid  bool
	}{
		{"instances: {}", false},
		{"instances: {one: {url: 'http://localhost', token_env: MACO_TEST_TOKEN}}", true},
		{"instances: {one: {url: 'http://localhost', token: x, token_env: MACO_TEST_TOKEN}}", false},
		{"instances: {one: {url: 'http://localhost', token_env: MACO_TEST_MISSING_TOKEN}}", false},
		{"instances: {one: {url: 'file:///tmp', token: x}}", false},
		{"instances: {one: {url: 'https://user:password@localhost', token: x}}", false},
	} {
		path := filepath.Join(t.TempDir(), "config.yml")
		if err := os.WriteFile(path, []byte(tc.config), 0600); err != nil {
			t.Fatal(err)
		}

		_, err := loadInstances(path)
		if (err == nil) != tc.valid {
			t.Errorf("%s: %v", tc.config, err)
		}
	}
}

func TestActionWaitsForReturnedJob(t *testing.T) {
	var requests []string
	cs, _ := connectTest(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer secret-test-token" {
			t.Error("missing token")
		}

		switch r.URL.Path {
		case "/api/vms/vm-1/backups":
			data, _ := io.ReadAll(r.Body)
			if len(data) != 0 || r.URL.RawQuery != "" {
				t.Error("wait leaked into API request")
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(202)
			_, _ = fmt.Fprint(w, `{"id":"returned-job","state":"pending"}`)
		case "/api/jobs/returned-job/wait":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprint(w, "event: update\ndata: {\"state\":\"active\"}\n\nevent: done\ndata: {\"id\":\"returned-job\",\"state\":\"succeeded\",\"result\":{\"backup\":\"saved\"}}\n\n")
		default:
			t.Errorf("unexpected request: %s", r.URL)
		}
	})
	result := call(t, cs, "createBackup", map[string]any{"instance": "test", "id": "vm-1", "wait": true})
	if result.IsError {
		t.Fatal(result)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, `"state":"succeeded"`) || !strings.Contains(text, `"backup":"saved"`) {
		t.Fatal(text)
	}

	if strings.Join(requests, ",") != "POST /api/vms/vm-1/backups,GET /api/jobs/returned-job/wait" {
		t.Fatal(requests)
	}
}

func TestActionWaitFailure(t *testing.T) {
	for _, mode := range []string{"failed-job", "disconnected", "missing-id"} {
		t.Run(mode, func(t *testing.T) {
			cs, _ := connectTest(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(202)
					if mode == "missing-id" {
						_, _ = fmt.Fprint(w, `{}`)
						return
					}

					_, _ = fmt.Fprint(w, `{"id":"job-42","state":"pending"}`)
					return
				}

				if mode == "missing-id" {
					t.Error("wait called without job ID")
				}

				if r.URL.Path != "/api/jobs/job-42/wait" {
					t.Error(r.URL.Path)
				}

				w.Header().Set("Content-Type", "text/event-stream")
				if mode == "failed-job" {
					_, _ = fmt.Fprint(w, "event: done\ndata: {\"id\":\"job-42\",\"state\":\"failed\",\"error\":\"backup failed\"}\n\n")
				}
			})
			result := call(t, cs, "createBackup", map[string]any{"instance": "test", "id": "vm", "wait": true})
			if !result.IsError {
				t.Fatal("expected wait failure")
			}

			text := result.Content[0].(*mcp.TextContent).Text
			if mode != "missing-id" && !strings.Contains(text, "job-42") {
				t.Fatal("lost submitted job ID: " + text)
			}
		})
	}
}

func TestActionWithoutWaitReturnsQueuedJob(t *testing.T) {
	cs, _ := connectTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Error("unexpected wait request")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(202)
		_, _ = fmt.Fprint(w, `{"id":"job-42","state":"pending"}`)
	})
	for _, args := range []map[string]any{
		{"instance": "test", "id": "vm"},
		{"instance": "test", "id": "vm", "wait": false},
	} {
		result := call(t, cs, "createBackup", args)
		if result.IsError || !strings.Contains(result.Content[0].(*mcp.TextContent).Text, `"state":"pending"`) {
			t.Fatal(result)
		}
	}
}
