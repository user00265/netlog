// Command netlog is the single binary: HTTP server, background workers, and the
// embedded Svelte SPA. There is no separate CLI surface.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"netlog"
	"netlog/internal/auth"
	"netlog/internal/callbook"
	"netlog/internal/config"
	"netlog/internal/db"
	"netlog/internal/dxcc"
	"netlog/internal/events"
	"netlog/internal/httpapi"
	"netlog/internal/logging"
	"netlog/internal/models"
	"netlog/internal/store"
	"netlog/internal/sync"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", configDefault(), "path to the YAML config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	logger, lvlErr := logging.New(logging.Options{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
		Output: os.Stdout,
	})
	slog.SetDefault(logger)
	if lvlErr != nil {
		logger.Warn("log level fallback", slog.String("error", lvlErr.Error()))
	}
	logger.Info("starting netlog", slog.String("addr", cfg.Server.Addr))

	// Root context cancelled on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sqldb, err := db.Open(cfg.Database.Path)
	if err != nil {
		return err
	}
	defer sqldb.Close()
	if err := db.Migrate(sqldb); err != nil {
		return err
	}
	st := store.New(sqldb)

	// DXCC dataset: load + schedule refresh in the background so startup is not
	// blocked by the initial download.
	dxccMgr := dxcc.NewManager(dxcc.Options{
		DataDir:      cfg.Data.Dir,
		RefreshEvery: time.Duration(cfg.Callbook.CTYRefreshDays) * 24 * time.Hour,
		HTTPClient:   &http.Client{Timeout: 60 * time.Second},
		Logger:       logger,
		Store:        st,
	})
	go func() {
		if err := dxccMgr.Start(ctx); err != nil {
			logger.Warn("dxcc dataset unavailable", slog.String("error", err.Error()))
		}
	}()

	resolver := callbook.NewResolver(cfg.Callbook, st, dxccMgr, logger,
		&http.Client{Timeout: cfg.Callbook.HTTPTimeout})

	oidcProvider, err := auth.NewOIDCProvider(ctx, cfg.OIDC)
	if err != nil {
		logger.Warn("oidc disabled: discovery failed", slog.String("error", err.Error()))
		oidcProvider = nil
	}

	spa, err := netlog.DistFS()
	if err != nil {
		return fmt.Errorf("load embedded frontend: %w", err)
	}

	hub := events.NewHub()

	server := httpapi.New(httpapi.Deps{
		BaseContext: ctx, // cancelled on SIGINT so SSE handlers release on shutdown
		Config:      cfg,
		Logger:      logger,
		Store:       st,
		Auth:        auth.NewService(st),
		Sessions:    auth.NewManager(st, cfg.Server.CookieSecure),
		OIDC:        oidcProvider,
		Sync:        sync.NewService(st, resolver, hub, logger),
		Callbook:    resolver,
		Events:      hub,
		SPA:         spa,
	})

	go pruneSessions(ctx, st, logger)

	httpServer := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      server.Handler(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Run the server until the context is cancelled, then shut down gracefully.
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("http server listening", slog.String("addr", cfg.Server.Addr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	}
}

// configDefault resolves the config path from NETLOG_CONFIG or falls back to
// ./config.yaml.
func configDefault() string {
	if p := os.Getenv("NETLOG_CONFIG"); p != "" {
		return p
	}
	return "config.yaml"
}

// pruneSessions periodically deletes expired sessions.
func pruneSessions(ctx context.Context, st *store.Store, logger *slog.Logger) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if n, err := st.DeleteExpiredSessions(ctx, models.Now()); err != nil {
				logger.WarnContext(ctx, "prune sessions failed", slog.String("error", err.Error()))
			} else if n > 0 {
				logger.DebugContext(ctx, "pruned expired sessions", slog.Int64("count", n))
			}
		}
	}
}
