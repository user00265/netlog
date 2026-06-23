// Package store is the data-access layer over the SQLite database. All SQL
// lives here so the rest of the application works with domain types only.
package store

import (
	"database/sql"
	"errors"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("not found")

// Store wraps the database handle and exposes typed query methods.
type Store struct {
	db *sql.DB
}

// New returns a Store backed by sqldb.
func New(sqldb *sql.DB) *Store {
	return &Store{db: sqldb}
}

// DB exposes the underlying handle for callers that need transactions or
// migrations.
func (s *Store) DB() *sql.DB { return s.db }
