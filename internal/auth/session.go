package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"netlog/internal/models"
	"netlog/internal/store"
)

// SessionCookie is the cookie name carrying the opaque session id.
const SessionCookie = "netlog_session"

// SessionTTL is how long a session remains valid.
const SessionTTL = 14 * 24 * time.Hour

// ErrNoSession indicates there is no valid authenticated session on the request.
var ErrNoSession = errors.New("no session")

// Manager issues and validates sessions and binds them to HTTP cookies.
type Manager struct {
	store  *store.Store
	secure bool
}

// NewManager returns a session Manager. secure controls the cookie Secure flag.
func NewManager(s *store.Store, secure bool) *Manager {
	return &Manager{store: s, secure: secure}
}

// Issue creates a session for userID and writes the session cookie.
func (m *Manager) Issue(ctx context.Context, w http.ResponseWriter, userID string) (models.Session, error) {
	token, err := NewToken()
	if err != nil {
		return models.Session{}, err
	}
	now := time.Now()
	sess := models.Session{
		ID:        token,
		UserID:    userID,
		CreatedAt: models.FormatTime(now),
		ExpiresAt: models.FormatTime(now.Add(SessionTTL)),
	}
	if err := m.store.CreateSession(ctx, sess); err != nil {
		return models.Session{}, err
	}
	//nolint:gosec // G124: HttpOnly+SameSite set; Secure is enabled via config for https deployments
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  now.Add(SessionTTL),
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: http.SameSiteLaxMode,
	})
	return sess, nil
}

// Authenticate resolves the session cookie to a user, validating expiry.
// Returns ErrNoSession when there is no valid session.
func (m *Manager) Authenticate(r *http.Request) (models.User, error) {
	cookie, err := r.Cookie(SessionCookie)
	if err != nil || cookie.Value == "" {
		return models.User{}, ErrNoSession
	}
	sess, err := m.store.GetSession(r.Context(), cookie.Value)
	if err != nil {
		return models.User{}, ErrNoSession
	}
	expires, err := models.ParseTime(sess.ExpiresAt)
	if err != nil || !time.Now().Before(expires) {
		// Expired or unparsable: best-effort cleanup, treat as no session.
		_ = m.store.DeleteSession(r.Context(), sess.ID)
		return models.User{}, ErrNoSession
	}
	user, err := m.store.GetUserByID(r.Context(), sess.UserID)
	if err != nil {
		return models.User{}, ErrNoSession
	}
	return user, nil
}

// Logout deletes the current session and clears the cookie.
func (m *Manager) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(SessionCookie); err == nil && cookie.Value != "" {
		_ = m.store.DeleteSession(r.Context(), cookie.Value)
	}
	//nolint:gosec // G124: HttpOnly+SameSite set; Secure is enabled via config for https deployments
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: http.SameSiteLaxMode,
	})
}
