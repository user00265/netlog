package dxcc

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"netlog/internal/models"
	"netlog/internal/store"
)

// Manager owns the cached cty.xml file, refreshes it on a schedule, and serves
// concurrent lookups against an atomically-swapped DB.
type Manager struct {
	url        string
	path       string
	refresh    time.Duration
	checkEvery time.Duration
	httpClient *http.Client
	logger     *slog.Logger
	store      *store.Store

	mu sync.RWMutex
	db *DB
}

// Options configure a Manager.
type Options struct {
	DataDir      string        // directory for the cached cty.xml.gz
	RefreshEvery time.Duration // how old the dataset may get before refresh
	HTTPClient   *http.Client
	Logger       *slog.Logger
	Store        *store.Store
}

// NewManager builds a Manager. It does not perform I/O until Start.
func NewManager(opts Options) *Manager {
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &Manager{
		url:        SourceURL,
		path:       filepath.Join(opts.DataDir, "cty.xml.gz"),
		refresh:    opts.RefreshEvery,
		checkEvery: 12 * time.Hour,
		httpClient: client,
		logger:     opts.Logger,
		store:      opts.Store,
	}
}

// Lookup resolves a callsign, or reports false if no dataset is loaded or no
// match is found.
func (m *Manager) Lookup(callsign string) (Result, bool) {
	m.mu.RLock()
	db := m.db
	m.mu.RUnlock()
	if db == nil {
		return Result{}, false
	}
	return db.Lookup(callsign)
}

// Loaded reports whether a dataset is currently loaded.
func (m *Manager) Loaded() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.db != nil
}

// Start loads the dataset (downloading if missing or stale) and then refreshes
// it on a schedule until ctx is cancelled. It blocks for the initial load and
// returns any error from it, but the periodic refresh loop is always started so
// a failed initial load (e.g. offline at boot) is retried rather than leaving
// DXCC permanently disabled until restart.
func (m *Manager) Start(ctx context.Context) error {
	err := m.ensureLoaded(ctx)
	go m.loop(ctx)
	return err
}

func (m *Manager) loop(ctx context.Context) {
	ticker := time.NewTicker(m.checkEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.ensureLoaded(ctx); err != nil {
				m.logger.ErrorContext(ctx, "dxcc refresh failed", slog.String("error", err.Error()))
			}
		}
	}
}

// ensureLoaded downloads the dataset when missing or stale, validates it, and
// swaps it in. A fresh download is parsed in a temp file and only installed over
// the existing good file once it parses successfully, so a corrupt response can
// never destroy the last-known-good dataset. The cty metadata (which gates the
// refresh cadence) is updated only on a genuinely successful download.
func (m *Manager) ensureLoaded(ctx context.Context) error {
	stale, err := m.stale(ctx)
	if err != nil {
		m.logger.WarnContext(ctx, "dxcc staleness check failed; attempting load", slog.String("error", err.Error()))
		stale = true
	}

	var refreshed bool
	if stale {
		if db, ok := m.tryDownload(ctx); ok {
			m.swap(db)
			if err := m.store.SetCtyMeta(ctx, models.Now(), db.Date(), db.EntityCount()); err != nil {
				m.logger.WarnContext(ctx, "recording cty meta failed", slog.String("error", err.Error()))
			}
			m.logger.InfoContext(ctx, "cty.xml dataset refreshed",
				slog.String("source_date", db.Date()), slog.Int("entities", db.EntityCount()))
			refreshed = true
		} else if !m.fileExists() {
			return errors.New("cty.xml unavailable: no usable dataset")
		}
	}

	// Not stale, or the refresh failed but an existing good file is present.
	if !refreshed {
		db, err := loadFile(m.path)
		if err != nil {
			return err
		}
		m.swap(db)
		m.logger.InfoContext(ctx, "cty.xml dataset loaded",
			slog.String("source_date", db.Date()), slog.Int("entities", db.EntityCount()))
	}
	return nil
}

// tryDownload fetches and validates a fresh dataset, installing it over the good
// file only if it parses. It returns the loaded DB and true on success.
func (m *Manager) tryDownload(ctx context.Context) (*DB, bool) {
	m.logger.InfoContext(ctx, "downloading cty.xml dataset", slog.String("url", m.url))
	tmp, err := download(ctx, m.httpClient, m.url, m.path)
	if err != nil {
		m.logger.WarnContext(ctx, "cty.xml download failed", slog.String("error", err.Error()))
		return nil, false
	}
	db, perr := loadFile(tmp)
	if perr != nil {
		_ = os.Remove(tmp)
		m.logger.WarnContext(ctx, "cty.xml download was invalid; keeping existing file", slog.String("error", perr.Error()))
		return nil, false
	}
	if rerr := os.Rename(tmp, m.path); rerr != nil {
		_ = os.Remove(tmp)
		m.logger.WarnContext(ctx, "installing cty.xml failed; keeping existing file", slog.String("error", rerr.Error()))
		return nil, false
	}
	return db, true
}

func (m *Manager) swap(db *DB) {
	m.mu.Lock()
	m.db = db
	m.mu.Unlock()
}

func (m *Manager) fileExists() bool {
	_, err := os.Stat(m.path)
	return err == nil
}

// stale reports whether the dataset should be refreshed: no file, no recorded
// download, or older than the refresh window.
func (m *Manager) stale(ctx context.Context) (bool, error) {
	if !m.fileExists() {
		return true, nil
	}
	meta, err := m.store.GetCtyMeta(ctx)
	if err != nil {
		return true, err
	}
	if meta.LastDownloadedAt == "" {
		return true, nil
	}
	// An unparseable timestamp yields the zero time, which is "very old" and so
	// correctly reads as stale.
	t, _ := models.ParseTime(meta.LastDownloadedAt)
	return time.Since(t) >= m.refresh, nil
}
