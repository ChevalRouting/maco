//go:build prod

package api

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/m-vinc/maco/pkg/config"
)

func embeddedServer(t *testing.T) *Server {
	t.Helper()

	static, err := fs.Sub(EmbeddedFS, "dist")
	if err != nil {
		t.Fatal(err)
	}

	entries, err := fs.ReadDir(static, ".")
	if err != nil || len(entries) == 0 {
		t.Fatalf("embedded dist is empty; build the UI before a prod build: %v", err)
	}

	paths, err := config.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	return New(paths, []byte("test-secret"), static, nil)
}

func TestEmbeddedUIServed(t *testing.T) {
	srv := embeddedServer(t)

	get := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}

	index := get("/")
	if index.Code != http.StatusOK {
		t.Fatalf("index status = %d", index.Code)
	}

	if ct := index.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("index content-type = %q", ct)
	}

	if !strings.Contains(index.Body.String(), "<script") {
		t.Fatal("index.html did not contain a script tag")
	}

	deep := get("/disks")
	if deep.Code != http.StatusOK || !strings.Contains(deep.Body.String(), "<script") {
		t.Fatalf("SPA fallback failed for /disks: status %d", deep.Code)
	}

	apiDocs := get("/api-docs")
	if apiDocs.Code != http.StatusOK || !strings.Contains(apiDocs.Body.String(), "<script") {
		t.Fatalf("SPA fallback failed for /api-docs: status %d", apiDocs.Code)
	}

	asset := findAsset(t, srv)
	res := get(asset)
	if res.Code != http.StatusOK {
		t.Fatalf("asset %s status = %d", asset, res.Code)
	}

	if ct := res.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Fatalf("asset %s content-type = %q", asset, ct)
	}

	miss := get("/api/does-not-exist")
	if miss.Code != http.StatusNotFound {
		t.Fatalf("unknown API route status = %d", miss.Code)
	}

	if !strings.Contains(miss.Body.String(), "API route not found") {
		t.Fatalf("unknown API body = %q", miss.Body.String())
	}
}

func findAsset(t *testing.T, srv *Server) string {
	t.Helper()

	var found string
	_ = fs.WalkDir(srv.static, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".js") {
			return nil
		}

		found = "/" + path
		return fs.SkipAll
	})

	if found == "" {
		t.Fatal("no built .js asset found in embedded dist")
	}

	return found
}
