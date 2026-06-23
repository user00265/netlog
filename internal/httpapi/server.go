// Package httpapi wires the HTTP surface: JSON API handlers, auth middleware,
// and the embedded SPA file server. The router uses the standard library
// net/http ServeMux (Go 1.22 method patterns).
package httpapi

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"

	"netlog/internal/auth"
	"netlog/internal/callbook"
	"netlog/internal/config"
	"netlog/internal/events"
	"netlog/internal/logging"
	"netlog/internal/store"
	"netlog/internal/sync"
)

// Deps are the dependencies the HTTP server needs.
type Deps struct {
	// BaseContext is cancelled when the process is shutting down. Long-lived
	// handlers (SSE) watch it so they return promptly during graceful shutdown
	// instead of blocking Server.Shutdown until its timeout. Defaults to
	// context.Background() (never cancelled) when nil.
	BaseContext context.Context
	Config      *config.Config
	Logger      *slog.Logger
	Store       *store.Store
	Auth        *auth.Service
	Sessions    *auth.Manager
	OIDC        *auth.OIDCProvider // may be nil when OIDC is disabled
	Sync        *sync.Service
	Callbook    *callbook.Resolver
	Events      *events.Hub
	SPA         fs.FS
}

// Server holds the HTTP dependencies and builds the router.
type Server struct {
	baseCtx  context.Context
	cfg      *config.Config
	logger   *slog.Logger
	store    *store.Store
	auth     *auth.Service
	sessions *auth.Manager
	oidc     *auth.OIDCProvider
	sync     *sync.Service
	callbook *callbook.Resolver
	events   *events.Hub
	spa      fs.FS
}

// New builds a Server from its dependencies.
func New(d Deps) *Server {
	baseCtx := d.BaseContext
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	return &Server{
		baseCtx:  baseCtx,
		cfg:      d.Config,
		logger:   d.Logger,
		store:    d.Store,
		auth:     d.Auth,
		sessions: d.Sessions,
		oidc:     d.OIDC,
		sync:     d.Sync,
		callbook: d.Callbook,
		events:   d.Events,
		spa:      d.SPA,
	}
}

// Handler returns the fully-wired HTTP handler with request logging applied.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Public auth endpoints.
	mux.HandleFunc("GET /api/bootstrap", s.handleBootstrap)
	mux.HandleFunc("POST /api/register", s.handleRegister)
	mux.HandleFunc("POST /api/login", s.handleLogin)
	mux.HandleFunc("POST /api/logout", s.handleLogout)
	mux.HandleFunc("GET /api/auth/oidc/start", s.handleOIDCStart)
	mux.HandleFunc("GET /api/auth/oidc/callback", s.handleOIDCCallback)

	// Authenticated endpoints.
	mux.Handle("GET /api/me", s.requireAuth(s.handleMe))
	mux.Handle("PATCH /api/account/profile", s.requireAuth(s.handleUpdateProfile))
	mux.Handle("POST /api/account/password", s.requireAuth(s.handleChangePassword))
	mux.Handle("GET /api/nets", s.requireAuth(s.handleListNets))
	mux.Handle("GET /api/nets/{id}", s.requireAuth(s.handleGetNet))
	mux.Handle("GET /api/callsign/{call}", s.requireAuth(s.handleGetCallsign))
	mux.Handle("POST /api/callsign/{call}/refresh", s.requireAuth(s.handleRefreshCallsign))
	mux.Handle("GET /api/sync", s.requireAuth(s.handlePull))
	mux.Handle("POST /api/sync", s.requireAuth(s.handlePush))
	mux.Handle("GET /api/events", s.requireAuth(s.handleEvents))

	// Admin endpoints.
	mux.Handle("GET /api/admin/users", s.requireAdmin(s.handleListUsers))
	mux.Handle("POST /api/admin/users", s.requireAdmin(s.handleCreateUser))

	// SPA (and 404 for unknown /api routes).
	mux.Handle("/", s.spaHandler())

	return logging.HTTPMiddleware(s.logger)(mux)
}

// requireAuth wraps a handler so only authenticated requests reach it.
func (s *Server) requireAuth(h http.HandlerFunc) http.Handler {
	return s.sessions.RequireAuth(s.unauthorized)(h)
}

// requireAdmin wraps a handler so only authenticated admins reach it.
func (s *Server) requireAdmin(h http.HandlerFunc) http.Handler {
	return s.sessions.RequireAuth(s.unauthorized)(auth.RequireAdmin(s.forbidden)(h))
}

func (s *Server) unauthorized(w http.ResponseWriter, _ *http.Request) {
	s.writeError(w, http.StatusUnauthorized, "authentication required")
}

func (s *Server) forbidden(w http.ResponseWriter, _ *http.Request) {
	s.writeError(w, http.StatusForbidden, "admin privileges required")
}
