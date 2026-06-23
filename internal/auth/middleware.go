package auth

import (
	"net/http"
)

// RequireAuth wraps a handler so it only runs for authenticated requests. The
// authenticated user is placed in the request context. unauthorized is called
// to write the rejection response.
func (m *Manager) RequireAuth(unauthorized http.HandlerFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := m.Authenticate(r)
			if err != nil {
				unauthorized(w, r)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
		})
	}
}

// RequireAdmin wraps a handler so it only runs for authenticated admins. It must
// be used after RequireAuth (it reads the user from context). forbidden writes
// the rejection response.
func RequireAdmin(forbidden http.HandlerFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFrom(r.Context())
			if !ok || !user.IsAdmin() {
				forbidden(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
