package httpapi

import (
	"errors"
	"net/http"

	"netlog/internal/models"
	"netlog/internal/store"
	"netlog/internal/validate"
)

// handleListNets returns all nets with NCS callsign and check-in counts, sorted
// by end time descending (active nets first).
func (s *Server) handleListNets(w http.ResponseWriter, r *http.Request) {
	nets, err := s.store.ListNets(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "could not list nets")
		return
	}
	if nets == nil {
		nets = []models.NetWithMeta{}
	}
	s.writeJSON(w, http.StatusOK, nets)
}

// netDetail is a net plus its check-ins for the net view.
type netDetail struct {
	models.Net
	CheckIns []models.CheckIn `json:"checkins"`
}

// handleGetNet returns a single (non-deleted) net with its check-ins.
func (s *Server) handleGetNet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validate.ValidID(id) {
		s.writeError(w, http.StatusBadRequest, "invalid net id")
		return
	}
	net, err := s.store.GetNet(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) || (err == nil && net.DeletedAt != nil) {
		s.writeError(w, http.StatusNotFound, "net not found")
		return
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "could not load net")
		return
	}

	checkins, err := s.store.ListCheckins(r.Context(), id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "could not load check-ins")
		return
	}
	if checkins == nil {
		checkins = []models.CheckIn{}
	}

	if u, err := s.store.GetUserByID(r.Context(), net.NCSUserID); err == nil {
		net.NCSCallsign = u.Callsign
	}

	s.writeJSON(w, http.StatusOK, netDetail{Net: net, CheckIns: checkins})
}
