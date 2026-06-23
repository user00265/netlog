package httpapi

import (
	"errors"
	"net/http"

	"netlog/internal/auth"
)

// handleUpdateProfile updates the authenticated user's editable profile.
func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	var in auth.ProfileInput
	if !s.decodeJSON(w, r, &in) {
		return
	}
	updated, err := s.auth.UpdateProfile(r.Context(), user.ID, in)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrCallsignTaken):
			s.writeError(w, http.StatusConflict, "callsign already registered")
		case errors.Is(err, auth.ErrEmailTaken):
			s.writeError(w, http.StatusConflict, "email already in use")
		default:
			s.writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	s.writeJSON(w, http.StatusOK, updated)
}

type changePasswordInput struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// handleChangePassword changes the authenticated user's password.
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	var in changePasswordInput
	if !s.decodeJSON(w, r, &in) {
		return
	}
	err := s.auth.ChangePassword(r.Context(), user.ID, in.CurrentPassword, in.NewPassword)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			s.writeError(w, http.StatusUnauthorized, "current password is incorrect")
			return
		}
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
