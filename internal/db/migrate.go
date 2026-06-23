package db

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate applies all pending migrations using the embedded migration files.
func Migrate(sqldb *sql.DB) error {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	// Quiet goose's own stdout logging; the application logs the outcome.
	goose.SetLogger(goose.NopLogger())
	if err := goose.Up(sqldb, "migrations"); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// Version returns the current migration version, for diagnostics.
func Version(sqldb *sql.DB) (int64, error) {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return 0, err
	}
	return goose.GetDBVersion(sqldb)
}
