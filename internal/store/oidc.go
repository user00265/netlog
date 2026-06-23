package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"netlog/internal/models"
)

// LinkOIDC associates an OIDC identity with a user.
func (s *Store) LinkOIDC(ctx context.Context, id models.OIDCIdentity) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO user_oidc (id, user_id, issuer, subject, created_at) VALUES (?, ?, ?, ?, ?)`,
		id.ID, id.UserID, id.Issuer, id.Subject, id.CreatedAt)
	if err != nil {
		return fmt.Errorf("link oidc: %w", err)
	}
	return nil
}

// GetUserByOIDC returns the user linked to the given issuer+subject, or
// ErrNotFound.
func (s *Store) GetUserByOIDC(ctx context.Context, issuer, subject string) (models.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users u
		 JOIN user_oidc o ON o.user_id = u.id
		 WHERE o.issuer = ? AND o.subject = ?`, issuer, subject)
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, ErrNotFound
	}
	return u, err
}

// OIDCLinksForUser lists the issuers a user has linked.
func (s *Store) OIDCLinksForUser(ctx context.Context, userID string) ([]models.OIDCIdentity, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, issuer, subject, created_at FROM user_oidc WHERE user_id = ?`, userID)
	if err != nil {
		return nil, fmt.Errorf("oidc links: %w", err)
	}
	defer rows.Close()

	var out []models.OIDCIdentity
	for rows.Next() {
		var i models.OIDCIdentity
		if err := rows.Scan(&i.ID, &i.UserID, &i.Issuer, &i.Subject, &i.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
