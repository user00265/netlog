package db

import (
	"path/filepath"
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.sqlite")
	sqldb, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer sqldb.Close()

	if err := Migrate(sqldb); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Migrations are idempotent: a second Up must be a no-op.
	if err := Migrate(sqldb); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}

	v, err := Version(sqldb)
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if v < 1 {
		t.Errorf("expected migration version >= 1, got %d", v)
	}

	// Foreign keys must be enforced (PRAGMA applied).
	var fk int
	if err := sqldb.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("query foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("expected foreign_keys ON, got %d", fk)
	}

	// Core tables exist.
	for _, table := range []string{"users", "user_oidc", "sessions", "nets", "checkins", "callsign_cache", "cty_meta"} {
		var name string
		err := sqldb.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("expected table %q to exist: %v", table, err)
		}
	}

	// The cty_meta singleton row was seeded.
	var count int
	if err := sqldb.QueryRow("SELECT entity_count FROM cty_meta WHERE id=1").Scan(&count); err != nil {
		t.Errorf("cty_meta seed row missing: %v", err)
	}
}
