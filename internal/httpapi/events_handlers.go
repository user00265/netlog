package httpapi

import (
	"fmt"
	"net/http"
	"time"
)

// sseHeartbeat is how often a keep-alive comment is sent so proxies and the
// client don't consider an idle connection dead.
const sseHeartbeat = 25 * time.Second

// handleEvents serves a Server-Sent Events stream. While the stream is open the
// client treats the backend as reachable; the server pushes "sync" events when
// data changes (so other sessions sync down) and "callsign" events when callbook
// enrichment finishes.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if s.events == nil {
		s.writeError(w, http.StatusNotFound, "events not available")
		return
	}

	rc := http.NewResponseController(w)
	// SSE connections are long-lived; clear the server's write deadline for this
	// one so the configured WriteTimeout doesn't tear it down.
	_ = rc.SetWriteDeadline(time.Time{})

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering (nginx)
	w.WriteHeader(http.StatusOK)

	ch := s.events.Subscribe()
	defer s.events.Unsubscribe(ch)

	// Open the stream so the client's onopen fires promptly.
	if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
		return
	}
	if err := rc.Flush(); err != nil {
		return
	}

	heartbeat := time.NewTicker(sseHeartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.baseCtx.Done():
			// The server is shutting down; release this long-lived handler so
			// Server.Shutdown can complete instead of waiting for the client.
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if _, err := fmt.Fprint(w, msg); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		}
	}
}
