package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

type openAPIDocument struct {
	OpenAPI string                                `json:"openapi"`
	Paths   map[string]map[string]json.RawMessage `json:"paths"`
}

func TestOpenAPIRoutes(t *testing.T) {
	s := &Server{}
	s.router = s.buildRouter()
	response := httptest.NewRecorder()
	s.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/docs/openapi.json", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("spec status: %d", response.Code)
	}

	var document openAPIDocument
	if err := json.Unmarshal(response.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.OpenAPI != "3.1.0" {
		t.Fatalf("OpenAPI version: %s", document.OpenAPI)
	}

	mounted := map[string]bool{}
	err := chi.Walk(s.router, func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if !strings.HasPrefix(route, "/api/") || route == "/api/docs/openapi.json" {
			return nil
		}
		key := strings.ToLower(method)
		mounted[key+" "+route] = true
		if _, ok := document.Paths[route][key]; !ok {
			t.Errorf("undocumented route: %s %s", method, route)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for route, methods := range document.Paths {
		for method := range methods {
			if !mounted[method+" "+route] {
				t.Errorf("unmounted operation: %s %s", method, route)
			}
		}
	}
}

func TestAgentContractIsSelfDescribing(t *testing.T) {
	s := &Server{}
	s.router = s.buildRouter()
	response := httptest.NewRecorder()
	s.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/docs/openapi.json", nil))
	var document map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	metadata, ok := document["x-maco"].(map[string]any)
	if !ok || metadata["version"] != float64(1) || metadata["instructions"] == "" {
		t.Fatal("missing agent discovery metadata")
	}
	operations := map[string]map[string]any{}
	for _, raw := range document["paths"].(map[string]any) {
		for _, entry := range raw.(map[string]any) {
			op := entry.(map[string]any)
			id := op["operationId"].(string)
			operations[id] = op
			meta, ok := op["x-maco"].(map[string]any)
			if !ok {
				t.Errorf("%s lacks explicit exposure metadata", id)
				continue
			}
			if meta["expose"] != true {
				continue
			}
			description, _ := op["description"].(string)
			if len(description) < 40 || description == id {
				t.Errorf("%s lacks usage guidance", id)
			}
			if _, ok := meta["readOnly"].(bool); !ok {
				t.Errorf("%s lacks read-only semantics", id)
			}
			responses := op["responses"].(map[string]any)
			if responses["202"] != nil && meta["wait"] == nil {
				t.Errorf("%s lacks job-wait linkage", id)
			}
			if meta["transport"] == "multipart" && meta["upload"] == nil {
				t.Errorf("%s lacks upload metadata", id)
			}
		}
	}
	for id, op := range operations {
		meta := op["x-maco"].(map[string]any)
		if wait, ok := meta["wait"].(map[string]any); ok {
			target := operations[wait["operationId"].(string)]
			if target == nil || target["x-maco"].(map[string]any)["stream"] == nil {
				t.Errorf("%s references invalid wait operation", id)
			}
		}
	}
	schemas := document["components"].(map[string]any)["schemas"].(map[string]any)
	create := schemas["engine.CreateVMParams"].(map[string]any)
	for field, raw := range create["properties"].(map[string]any) {
		schema := raw.(map[string]any)
		if ref, ok := schema["$ref"].(string); ok {
			schema = schemas[strings.TrimPrefix(ref, "#/components/schemas/")].(map[string]any)
		}
		if schema["description"] == nil {
			t.Errorf("createVM field %s lacks documentation", field)
		}
	}
}
