package api

import (
	"embed"
	"net/http"
)

//go:embed docs/swagger.json
var openAPIFiles embed.FS

func (s *Server) serveOpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	http.ServeFileFS(w, r, openAPIFiles, "docs/swagger.json")
}
