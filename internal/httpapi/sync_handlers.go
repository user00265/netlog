package httpapi

import (
	"net/http"

	"netlog/internal/auth"
	"netlog/internal/models"
	"netlog/internal/sync"
)

// handlePull returns records changed since the given timestamp (query "since";
// empty means a full sync). Used by the SPA to sync down on load/reconnect.
func (s *Server) handlePull(w http.ResponseWriter, r *http.Request) {
	// Only accept a well-formed timestamp cursor; anything else falls back to a
	// full sync rather than passing untrusted text downstream.
	since := r.URL.Query().Get("since")
	if since != "" {
		if _, err := models.ParseTime(since); err != nil {
			since = ""
		}
	}
	resp, err := s.sync.Pull(r.Context(), since)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "sync pull failed")
		return
	}
	s.writeJSON(w, http.StatusOK, resp)
}

// handlePush applies a batch of client changes (the outbox flush) and returns
// per-change results with the authoritative server-side records.
func (s *Server) handlePush(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	var req sync.PushRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}
	resp := s.sync.Push(r.Context(), user, req)
	s.writeJSON(w, http.StatusOK, resp)
}
