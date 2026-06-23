package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"netlog/internal/models"
)

const userColumns = `id, callsign, first_name, last_name, email, password_hash, role, timezone, time_format, created_at, updated_at`

func scanUser(row interface{ Scan(...any) error }) (models.User, error) {
	var u models.User
	var pw sql.NullString
	err := row.Scan(&u.ID, &u.Callsign, &u.FirstName, &u.LastName, &u.Email, &pw, &u.Role,
		&u.Timezone, &u.TimeFormat, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return models.User{}, err
	}
	u.PasswordHash = pw.String
	return u, nil
}

// CountUsers returns the number of accounts. Zero means first-run.
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}

// CreateUser inserts a new account.
func (s *Store) CreateUser(ctx context.Context, u models.User) error {
	var pw any
	if u.PasswordHash != "" {
		pw = u.PasswordHash
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (`+userColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Callsign, u.FirstName, u.LastName, u.Email, pw, u.Role,
		u.Timezone, u.TimeFormat, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// UpdateProfile updates a user's editable profile fields.
func (s *Store) UpdateProfile(ctx context.Context, u models.User) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET callsign = ?, first_name = ?, last_name = ?, email = ?,
		 timezone = ?, time_format = ?, updated_at = ? WHERE id = ?`,
		u.Callsign, u.FirstName, u.LastName, u.Email, u.Timezone, u.TimeFormat, u.UpdatedAt, u.ID)
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	return nil
}

// GetUserByID returns a user by id, or ErrNotFound.
func (s *Store) GetUserByID(ctx context.Context, id string) (models.User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id = ?`, id)
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, ErrNotFound
	}
	return u, err
}

// GetUserByCallsign returns a user by callsign (case-sensitive; callers should
// pass the normalized uppercase form), or ErrNotFound.
func (s *Store) GetUserByCallsign(ctx context.Context, callsign string) (models.User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE callsign = ?`, callsign)
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, ErrNotFound
	}
	return u, err
}

// ListUsers returns all users ordered by callsign.
func (s *Store) ListUsers(ctx context.Context) ([]models.User, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+userColumns+` FROM users ORDER BY callsign`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// SetPassword updates a user's password hash and updated_at.
func (s *Store) SetPassword(ctx context.Context, userID, hash, updatedAt string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`, hash, updatedAt, userID)
	if err != nil {
		return fmt.Errorf("set password: %w", err)
	}
	return nil
}

// GetUserByEmail returns a user by email (used for OIDC account linking), or
// ErrNotFound.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE email = ? COLLATE NOCASE`, email)
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, ErrNotFound
	}
	return u, err
}

// CreateFirstAdmin atomically inserts u only if no accounts exist yet. It
// returns false (without error) when an account already exists, closing the
// first-admin registration window even under concurrent requests.
func (s *Store) CreateFirstAdmin(ctx context.Context, u models.User) (bool, error) {
	var pw any
	if u.PasswordHash != "" {
		pw = u.PasswordHash
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO users (`+userColumns+`)
		 SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		 WHERE (SELECT COUNT(*) FROM users) = 0`,
		u.ID, u.Callsign, u.FirstName, u.LastName, u.Email, pw, u.Role,
		u.Timezone, u.TimeFormat, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		return false, fmt.Errorf("create first admin: %w", err)
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}
