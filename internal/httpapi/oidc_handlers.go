package httpapi

import (
	"errors"
	"net/http"

	"netlog/internal/auth"
	"netlog/internal/models"
	"netlog/internal/store"
)

const (
	oidcStateCookie = "netlog_oidc_state"
	oidcNonceCookie = "netlog_oidc_nonce"
	oidcPKCECookie  = "netlog_oidc_pkce_verifier"
)

// handleOIDCStart begins the OIDC auth-code flow, stashing CSRF state and replay
// nonce in short-lived cookies.
func (s *Server) handleOIDCStart(w http.ResponseWriter, r *http.Request) {
	if s.oidc == nil {
		s.writeError(w, http.StatusNotFound, "OIDC is not enabled")
		return
	}
	state, err1 := auth.NewToken()
	nonce, err2 := auth.NewToken()
	var pkceVerifier string
	var err3 error
	if s.oidc.RequiresPKCE() {
		pkceVerifier, err3 = auth.NewToken()
	}
	if err1 != nil || err2 != nil || err3 != nil {
		s.writeError(w, http.StatusInternalServerError, "could not start OIDC flow")
		return
	}
	s.setShortCookie(w, oidcStateCookie, state)
	s.setShortCookie(w, oidcNonceCookie, nonce)
	if s.oidc.RequiresPKCE() {
		s.setShortCookie(w, oidcPKCECookie, pkceVerifier)
	}
	http.Redirect(w, r, s.oidc.AuthCodeURL(state, nonce, pkceVerifier), http.StatusFound)
}

// handleOIDCCallback completes the flow: it verifies state, exchanges the code,
// and either logs in a linked account or links the identity to an existing
// account (by current session or matching email). Accounts are never
// auto-created — registration is admin-only.
func (s *Server) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	if s.oidc == nil {
		s.writeError(w, http.StatusNotFound, "OIDC is not enabled")
		return
	}
	defer s.clearShortCookie(w, oidcStateCookie)
	defer s.clearShortCookie(w, oidcNonceCookie)
	defer s.clearShortCookie(w, oidcPKCECookie)

	stateCookie, err := r.Cookie(oidcStateCookie)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
		s.writeError(w, http.StatusBadRequest, "invalid OIDC state")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		s.writeError(w, http.StatusBadRequest, "missing authorization code")
		return
	}
	var nonce string
	if c, err := r.Cookie(oidcNonceCookie); err == nil {
		nonce = c.Value
	}
	var pkceVerifier string
	if s.oidc.RequiresPKCE() {
		c, err := r.Cookie(oidcPKCECookie)
		if err != nil || c.Value == "" {
			s.writeError(w, http.StatusBadRequest, "invalid OIDC PKCE verifier")
			return
		}
		pkceVerifier = c.Value
	}

	claims, err := s.oidc.Exchange(r.Context(), code, nonce, pkceVerifier)
	if err != nil {
		s.logger.WarnContext(r.Context(), "oidc exchange failed", "error", err.Error())
		http.Redirect(w, r, "/login?error=oidc", http.StatusFound)
		return
	}

	// Already linked → log in.
	if user, err := s.store.GetUserByOIDC(r.Context(), claims.Issuer, claims.Subject); err == nil {
		s.issueAndRedirect(w, r, user.ID)
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		s.writeError(w, http.StatusInternalServerError, "oidc lookup failed")
		return
	}

	// Link to the currently logged-in user, if any.
	if current, err := s.sessions.Authenticate(r); err == nil {
		if s.linkOIDC(r, current.ID, claims) {
			http.Redirect(w, r, "/", http.StatusFound)
		} else {
			http.Redirect(w, r, "/settings?error=oidc_link", http.StatusFound)
		}
		return
	}

	// Otherwise link by matching email to a pre-created account, then log in.
	// Only a verified email may auto-link, otherwise an IdP that lets users set
	// an arbitrary email could be used to take over an account.
	if claims.Email != "" && claims.EmailVerified {
		if user, err := s.store.GetUserByEmail(r.Context(), claims.Email); err == nil {
			if s.linkOIDC(r, user.ID, claims) {
				s.issueAndRedirect(w, r, user.ID)
				return
			}
		}
	}

	http.Redirect(w, r, "/login?error=nouser", http.StatusFound)
}

func (s *Server) linkOIDC(r *http.Request, userID string, claims auth.OIDCClaims) bool {
	err := s.store.LinkOIDC(r.Context(), models.OIDCIdentity{
		ID:        auth.NewID(),
		UserID:    userID,
		Issuer:    claims.Issuer,
		Subject:   claims.Subject,
		CreatedAt: models.Now(),
	})
	if err != nil {
		s.logger.WarnContext(r.Context(), "oidc link failed", "error", err.Error())
		return false
	}
	return true
}

func (s *Server) issueAndRedirect(w http.ResponseWriter, r *http.Request, userID string) {
	if _, err := s.sessions.Issue(r.Context(), w, userID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "could not start session")
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) setShortCookie(w http.ResponseWriter, name, value string) {
	//nolint:gosec // G124: HttpOnly+SameSite set; Secure is enabled via config for https deployments
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/api/auth/oidc",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   s.cfg.Server.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) clearShortCookie(w http.ResponseWriter, name string) {
	//nolint:gosec // G124: HttpOnly+SameSite set; Secure is enabled via config for https deployments
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/api/auth/oidc",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cfg.Server.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
