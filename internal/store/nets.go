package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"netlog/internal/models"
)

const netColumns = `id, name, net_date, ncs_user_id, status, start_at, end_at, notes, created_at, updated_at, deleted_at`

func scanNet(row interface{ Scan(...any) error }) (models.Net, error) {
	var n models.Net
	var start, end, deleted sql.NullString
	err := row.Scan(&n.ID, &n.Name, &n.NetDate, &n.NCSUserID, &n.Status, &start, &end, &n.Notes, &n.CreatedAt, &n.UpdatedAt, &deleted)
	if err != nil {
		return models.Net{}, err
	}
	n.StartAt = nullToPtr(start)
	n.EndAt = nullToPtr(end)
	n.DeletedAt = nullToPtr(deleted)
	return n, nil
}

func nullToPtr(s sql.NullString) *string {
	if s.Valid {
		v := s.String
		return &v
	}
	return nil
}

// CreateNet inserts a new net.
func (s *Store) CreateNet(ctx context.Context, n models.Net) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO nets (`+netColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		n.ID, n.Name, n.NetDate, n.NCSUserID, n.Status, ptrToNull(n.StartAt), ptrToNull(n.EndAt),
		n.Notes, n.CreatedAt, n.UpdatedAt, ptrToNull(n.DeletedAt))
	if err != nil {
		return fmt.Errorf("create net: %w", err)
	}
	return nil
}

// UpdateNet writes all mutable fields of an existing net.
func (s *Store) UpdateNet(ctx context.Context, n models.Net) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE nets SET name=?, net_date=?, status=?, start_at=?, end_at=?, notes=?, updated_at=?, deleted_at=? WHERE id=?`,
		n.Name, n.NetDate, n.Status, ptrToNull(n.StartAt), ptrToNull(n.EndAt), n.Notes, n.UpdatedAt, ptrToNull(n.DeletedAt), n.ID)
	if err != nil {
		return fmt.Errorf("update net: %w", err)
	}
	return nil
}

// GetNet returns a net by id (including soft-deleted), or ErrNotFound.
func (s *Store) GetNet(ctx context.Context, id string) (models.Net, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+netColumns+` FROM nets WHERE id = ?`, id)
	n, err := scanNet(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Net{}, ErrNotFound
	}
	return n, err
}

// ListNets returns non-deleted nets with NCS callsign and check-in counts,
// sorted by end date/time descending (active nets — no end time — first).
func (s *Store) ListNets(ctx context.Context) ([]models.NetWithMeta, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT n.id, n.name, n.net_date, n.ncs_user_id, n.status, n.start_at, n.end_at, n.notes,
		       n.created_at, n.updated_at, n.deleted_at,
		       COALESCE(u.callsign, ''),
		       (SELECT COUNT(*) FROM checkins c WHERE c.net_id = n.id AND c.deleted_at IS NULL)
		FROM nets n
		LEFT JOIN users u ON u.id = n.ncs_user_id
		WHERE n.deleted_at IS NULL
		ORDER BY n.end_at IS NULL DESC, n.end_at DESC, n.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list nets: %w", err)
	}
	defer rows.Close()

	var out []models.NetWithMeta
	for rows.Next() {
		var n models.Net
		var start, end, deleted sql.NullString
		var m models.NetWithMeta
		if err := rows.Scan(&n.ID, &n.Name, &n.NetDate, &n.NCSUserID, &n.Status, &start, &end, &n.Notes,
			&n.CreatedAt, &n.UpdatedAt, &deleted, &n.NCSCallsign, &m.CheckInCount); err != nil {
			return nil, err
		}
		n.StartAt = nullToPtr(start)
		n.EndAt = nullToPtr(end)
		n.DeletedAt = nullToPtr(deleted)
		m.Net = n
		out = append(out, m)
	}
	return out, rows.Err()
}

// ChangedNetsSince returns nets whose updated_at is strictly greater than since,
// including tombstones (soft-deleted), for the sync-down protocol. The NCS
// callsign is joined in so clients can display it offline.
func (s *Store) ChangedNetsSince(ctx context.Context, since string) ([]models.Net, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT n.id, n.name, n.net_date, n.ncs_user_id, n.status, n.start_at, n.end_at, n.notes,
		       n.created_at, n.updated_at, n.deleted_at, COALESCE(u.callsign, '')
		FROM nets n LEFT JOIN users u ON u.id = n.ncs_user_id
		WHERE n.updated_at > ? ORDER BY n.updated_at`, since)
	if err != nil {
		return nil, fmt.Errorf("changed nets: %w", err)
	}
	defer rows.Close()

	var out []models.Net
	for rows.Next() {
		var n models.Net
		var start, end, deleted sql.NullString
		if err := rows.Scan(&n.ID, &n.Name, &n.NetDate, &n.NCSUserID, &n.Status, &start, &end, &n.Notes,
			&n.CreatedAt, &n.UpdatedAt, &deleted, &n.NCSCallsign); err != nil {
			return nil, err
		}
		n.StartAt = nullToPtr(start)
		n.EndAt = nullToPtr(end)
		n.DeletedAt = nullToPtr(deleted)
		out = append(out, n)
	}
	return out, rows.Err()
}

func ptrToNull(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}
