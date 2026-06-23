package httpapi

import (
	"errors"
	"net/http"

	"netlog/internal/auth"
	"netlog/internal/config"
	"netlog/internal/validate"
)

// handleBootstrap reports whether the forced first-admin registration is needed
// and whether OIDC login is available, so the SPA can render the right screen.
func (s *Server) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	need, err := s.auth.NeedsFirstAdmin(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "bootstrap failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"needsFirstAdmin": need,
		"oidcEnabled":     s.oidc != nil,
		"version":         config.Version,
		"commit":          config.Revision,
	})
}

// handleRegister creates the very first account (an admin). It is closed once
// any account exists.
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var in auth.AccountInput
	if !s.decodeJSON(w, r, &in) {
		return
	}
	user, err := s.auth.RegisterFirstAdmin(r.Context(), in)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrRegistrationClosed):
			s.writeError(w, http.StatusForbidden, "registration is closed")
		case errors.Is(err, auth.ErrCallsignTaken):
			s.writeError(w, http.StatusConflict, "callsign already registered")
		default:
			s.writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	if _, err := s.sessions.Issue(r.Context(), w, user.ID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "could not start session")
		return
	}
	s.writeJSON(w, http.StatusCreated, user)
}

type loginInput struct {
	Callsign string `json:"callsign"`
	Password string `json:"password"`
}

// handleLogin authenticates a callsign/password and starts a session.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in loginInput
	if !s.decodeJSON(w, r, &in) {
		return
	}
	user, err := s.auth.Authenticate(r.Context(), in.Callsign, in.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			s.writeError(w, http.StatusUnauthorized, "invalid callsign or password")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "login failed")
		return
	}
	if _, err := s.sessions.Issue(r.Context(), w, user.ID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "could not start session")
		return
	}
	s.writeJSON(w, http.StatusOK, user)
}

// handleLogout ends the current session.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.sessions.Logout(w, r)
	w.WriteHeader(http.StatusNoContent)
}

// handleMe returns the authenticated user.
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	s.writeJSON(w, http.StatusOK, user)
}

// handleListUsers returns all accounts (admin only).
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "could not list users")
		return
	}
	s.writeJSON(w, http.StatusOK, users)
}

// handleCreateUser creates a non-admin account (admin only).
func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var in auth.AccountInput
	if !s.decodeJSON(w, r, &in) {
		return
	}
	user, err := s.auth.CreateUser(r.Context(), in)
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
	s.writeJSON(w, http.StatusCreated, user)
}

// validCallsignParam normalizes and validates a {call} path value.
func validCallsignParam(raw string) (string, bool) {
	c := validate.NormalizeCallsign(raw)
	if !validate.ValidCallsign(c) {
		return "", false
	}
	return c, true
}
