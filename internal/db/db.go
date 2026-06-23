// Package db opens the SQLite database and applies embedded migrations.
//
// We use the pure-Go modernc.org/sqlite driver (no cgo) so the binary
// cross-compiles to a single static executable.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, registered as "sqlite"
)

// Open creates (if needed) and opens the SQLite database at path. Standard
// pragmas (WAL, busy timeout, foreign keys, synchronous=NORMAL) are applied to
// every pooled connection via the DSN.
func Open(path string) (*sql.DB, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("create data dir: %w", err)
		}
	}

	// Apply pragmas on every new connection via the DSN so pooled connections
	// are configured consistently.
	dsn := path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(ON)&_pragma=synchronous(NORMAL)"

	sqldb, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqldb.PingContext(ctx); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return sqldb, nil
}
