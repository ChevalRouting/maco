package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-vinc/maco/pkg/config"
	"github.com/m-vinc/maco/pkg/engine"
)

func TestTemplateEndpoints(t *testing.T) {
	paths, err := config.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	s := New(paths, []byte("secret"), nil, nil)
	token, err := issueAPITestToken(t, s)
	if err != nil {
		t.Fatal(err)
	}

	do := func(method, path string, body any) *httptest.ResponseRecorder {
		var reader *bytes.Reader
		if body != nil {
			raw, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}

			reader = bytes.NewReader(raw)
		} else {
			reader = bytes.NewReader(nil)
		}

		r := httptest.NewRequest(method, path, reader)
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		return w
	}

	create := do(http.MethodPost, "/api/templates", TemplateRequest{
		Name:        "flatcar-profile",
		Description: "nginx host",
		Spec:        engine.CreateVMParams{Image: "flatcar-stable-arm64", CPUs: 2, MemoryMiB: 2048, DiskSizeGiB: 20},
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", create.Code, create.Body.String())
	}

	var created engine.Template
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	if created.ID == "" || created.Spec.Name != "" {
		t.Fatalf("unexpected created: %+v", created)
	}

	list := do(http.MethodGet, "/api/templates", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list: %d", list.Code)
	}

	var all []engine.Template
	if err := json.Unmarshal(list.Body.Bytes(), &all); err != nil || len(all) != 1 {
		t.Fatalf("list body: %v %s", err, list.Body.String())
	}

	get := do(http.MethodGet, "/api/templates/"+created.ID, nil)
	if get.Code != http.StatusOK {
		t.Fatalf("get: %d", get.Code)
	}

	if bad := do(http.MethodPost, "/api/templates/"+created.ID+"/instantiate", map[string]any{"name": "bad name!"}); bad.Code != http.StatusBadRequest {
		t.Fatalf("instantiate invalid name: %d %s", bad.Code, bad.Body.String())
	}

	missing := do(http.MethodPost, "/api/templates/00000000-0000-0000-0000-000000000000/instantiate", map[string]any{"name": "web-01"})
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("instantiate missing template: %d", missing.Code)
	}

	del := do(http.MethodDelete, "/api/templates/"+created.ID, nil)
	if del.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", del.Code)
	}

	if again := do(http.MethodGet, "/api/templates/"+created.ID, nil); again.Code != http.StatusNotFound {
		t.Fatalf("get after delete: %d", again.Code)
	}
}
