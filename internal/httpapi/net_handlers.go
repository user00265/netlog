package httpapi

import (
	"context"
	"errors"
	"net/http"

	"netlog/internal/auth"
	"netlog/internal/models"
	"netlog/internal/store"
	"netlog/internal/validate"
)

// handleListNets returns all nets with NCS callsign and check-in counts, sorted
// by end time descending (active nets first). canManage is per requesting user.
func (s *Server) handleListNets(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	nets, err := s.store.ListNets(r.Context(), user.ID, user.IsAdmin())
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
	net.CanManage = s.canManageNet(r.Context(), net)

	s.writeJSON(w, http.StatusOK, netDetail{Net: net, CheckIns: checkins})
}

// canManageNet reports whether the request's user may edit the net: an admin,
// any operator while the net is unassigned (no NCS yet), or a member of its
// controller set once claimed.
func (s *Server) canManageNet(ctx context.Context, net models.Net) bool {
	user, ok := auth.UserFrom(ctx)
	if !ok {
		return false
	}
	if user.IsAdmin() || net.NCSUserID == "" {
		return true
	}
	can, err := s.store.IsNetController(ctx, net.ID, user.ID)
	if err != nil {
		s.logger.ErrorContext(ctx, "net controller check failed", "net", net.ID, "error", err)
		return false
	}
	return can
}

// reassignInput carries the callsign of the new NCS for a hand-off.
type reassignInput struct {
	Callsign string `json:"callsign"`
}

// handleReassignNCS hands a net's NCS role to another operator by callsign. The
// caller must already manage the net (a controller or an admin); the new NCS is
// added to the controller set and becomes the displayed NCS while prior
// controllers retain access. Online-only by nature (it resolves a callsign to an
// account), so the SPA disables it while offline.
func (s *Server) handleReassignNCS(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validate.ValidID(id) {
		s.writeError(w, http.StatusBadRequest, "invalid net id")
		return
	}
	var in reassignInput
	if !s.decodeJSON(w, r, &in) {
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
	if !s.canManageNet(r.Context(), net) {
		s.writeError(w, http.StatusForbidden, "you do not control this net")
		return
	}
	// A closed net is read-only: no NCS hand-off after it ends. The SPA hides the
	// button, but enforce it here too so a stale client or direct call can't edit.
	if net.Status == models.NetClosed {
		s.writeError(w, http.StatusConflict, "net is closed")
		return
	}

	callsign, ok := validCallsignParam(in.Callsign)
	if !ok {
		s.writeError(w, http.StatusBadRequest, "invalid callsign")
		return
	}
	target, err := s.store.GetUserByCallsign(r.Context(), callsign)
	if errors.Is(err, store.ErrNotFound) {
		s.writeError(w, http.StatusNotFound, "no operator with that callsign")
		return
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "could not look up operator")
		return
	}

	now := models.Now()
	if err := s.store.AddNetController(r.Context(), id, target.ID, now); err != nil {
		s.writeError(w, http.StatusInternalServerError, "could not reassign NCS")
		return
	}
	if err := s.store.SetNetNCS(r.Context(), id, target.ID, now); err != nil {
		s.writeError(w, http.StatusInternalServerError, "could not reassign NCS")
		return
	}
	// Tell connected clients to sync down the new NCS / controller access.
	s.events.Broadcast("sync", "1")

	net.NCSUserID = target.ID
	net.NCSCallsign = target.Callsign
	net.UpdatedAt = now
	net.CanManage = s.canManageNet(r.Context(), net)
	s.writeJSON(w, http.StatusOK, net)
}
