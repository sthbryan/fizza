package httpapi

import (
	"net/http"

	"github.com/fizza/fizza/internal/db"
)

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	project := r.URL.Query().Get("project")
	board := r.URL.Query().Get("board")

	stats, err := db.GetStats(ctx, s.svc.DB(), project, board)
	if err != nil {
		respondError(w, err)
		return
	}
	writeOK(w, http.StatusOK, stats)
}
