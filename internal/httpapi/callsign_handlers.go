package httpapi

import (
	"errors"
	"net/http"

	"netlog/internal/callbook"
	"netlog/internal/store"
)

// handleGetCallsign returns cached callbook/DXCC data for a callsign without
// contacting the providers.
func (s *Server) handleGetCallsign(w http.ResponseWriter, r *http.Request) {
	call, ok := validCallsignParam(r.PathValue("call"))
	if !ok {
		s.writeError(w, http.StatusBadRequest, "invalid callsign")
		return
	}
	data, err := s.callbook.Cached(r.Context(), call)
	if errors.Is(err, store.ErrNotFound) {
		s.writeError(w, http.StatusNotFound, "no cached data for callsign")
		return
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	s.writeJSON(w, http.StatusOK, data)
}

// handleRefreshCallsign forces a fresh callbook lookup (the per-callsign refresh
// button) and returns the updated data.
func (s *Server) handleRefreshCallsign(w http.ResponseWriter, r *http.Request) {
	call, ok := validCallsignParam(r.PathValue("call"))
	if !ok {
		s.writeError(w, http.StatusBadRequest, "invalid callsign")
		return
	}
	data, err := s.callbook.Lookup(r.Context(), call, true)
	if errors.Is(err, callbook.ErrCallsignNotFound) {
		s.writeError(w, http.StatusNotFound, "callsign not found in any callbook")
		return
	}
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "callbook lookup failed")
		return
	}
	s.writeJSON(w, http.StatusOK, data)
}
