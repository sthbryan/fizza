package httpapi

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed static/*
var staticFS embed.FS

func (s *Server) mountWeb() {
	if appFS, err := fs.Sub(staticFS, "static/app"); err == nil {
		if _, err := fs.Stat(appFS, "index.html"); err == nil {
			fileServer := http.FileServer(http.FS(appFS))
			s.mux.Handle("GET /assets/", fileServer)
			s.mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))
		}
	}

	s.mux.HandleFunc("GET /{$}", s.handleIndex)
	s.mux.HandleFunc("GET /projects", s.handleIndex)
	s.mux.HandleFunc("GET /stats", s.handleIndex)
	s.mux.HandleFunc("GET /p/{project}/b/{board}", s.handleIndex)
	s.mux.HandleFunc("GET /p/{project}/b/{board}/archived", s.handleIndex)
	s.mux.HandleFunc("GET /{path...}", s.handleSPAFallback)
}

func (s *Server) handleSPAFallback(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/healthz" || strings.HasPrefix(path, "/v1/") {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "not found")
		return
	}
	if strings.HasPrefix(path, "/assets/") || strings.HasPrefix(path, "/static/") {
		http.NotFound(w, r)
		return
	}
	s.handleIndex(w, r)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := staticFS.ReadFile("static/app/index.html")
	if err != nil {
		data, err = staticFS.ReadFile("static/fallback.html")
		if err != nil {
			http.Error(w, "web UI not built; run: make web", http.StatusInternalServerError)
			return
		}
	}

	html := string(data)
	if !strings.Contains(html, `src="/assets/`) && strings.Contains(html, `src="./assets/`) {
		html = strings.ReplaceAll(html, `src="./assets/`, `src="/assets/`)
		html = strings.ReplaceAll(html, `href="./assets/`, `href="/assets/`)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write([]byte(html))
}
