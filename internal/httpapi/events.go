package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/fizza/fizza/internal/db"
)

func (s *Server) handleEventsSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	if rc := http.NewResponseController(w); rc != nil {
		_ = rc.SetWriteDeadline(time.Time{})
	}

	ctx := r.Context()
	after, err := parseAfterCursor(r)
	if err != nil {
		http.Error(w, "invalid after/Last-Event-ID", http.StatusBadRequest)
		return
	}
	if after == 0 {

		maxID, err := db.MaxEventID(ctx, s.svc.DB())
		if err != nil {
			http.Error(w, "events unavailable", http.StatusInternalServerError)
			return
		}
		after = maxID
	}

	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	poll := time.NewTicker(400 * time.Millisecond)
	defer poll.Stop()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-poll.C:
			events, err := db.EventsAfter(ctx, s.svc.DB(), after, 100)
			if err != nil {
				continue
			}
			for _, ev := range events {
				payload, err := json.Marshal(ev)
				if err != nil {
					continue
				}
				if _, err := fmt.Fprintf(w, "id: %d\nevent: change\ndata: %s\n\n", ev.ID, payload); err != nil {
					return
				}
				after = ev.ID
				flusher.Flush()
			}
		}
	}
}

func parseAfterCursor(r *http.Request) (int64, error) {
	if v := r.Header.Get("Last-Event-ID"); v != "" {
		return strconv.ParseInt(v, 10, 64)
	}
	if v := r.URL.Query().Get("after"); v != "" {
		return strconv.ParseInt(v, 10, 64)
	}
	return 0, nil
}
