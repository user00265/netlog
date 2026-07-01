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
	var ncs, start, end, deleted sql.NullString
	err := row.Scan(&n.ID, &n.Name, &n.NetDate, &ncs, &n.Status, &start, &end, &n.Notes, &n.CreatedAt, &n.UpdatedAt, &deleted)
	if err != nil {
		return models.Net{}, err
	}
	n.NCSUserID = ncs.String // "" when unassigned (NULL)
	n.StartAt = nullToPtr(start)
	n.EndAt = nullToPtr(end)
	n.DeletedAt = nullToPtr(deleted)
	return n, nil
}

// nullIfEmpty maps an empty NCS id to SQL NULL (an unassigned net).
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
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
		n.ID, n.Name, n.NetDate, nullIfEmpty(n.NCSUserID), n.Status, ptrToNull(n.StartAt), ptrToNull(n.EndAt),
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

// canManageExpr is a SELECT-list fragment computing models.Net.CanManage for a
// requester: admins manage everything; anyone may manage a still-unassigned net
// (no NCS yet); otherwise membership in net_controllers. Bind (isAdmin int,
// requesterID) in that order, before any other parameters.
const canManageExpr = `(? OR COALESCE(n.ncs_user_id, '') = '' OR EXISTS(SELECT 1 FROM net_controllers nc WHERE nc.net_id = n.id AND nc.user_id = ?))`

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ListNets returns non-deleted nets with NCS callsign and check-in counts,
// sorted by end date/time descending (active nets — no end time — first).
// canManage is computed for requesterID (admins manage all).
func (s *Store) ListNets(ctx context.Context, requesterID string, isAdmin bool) ([]models.NetWithMeta, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT n.id, n.name, n.net_date, n.ncs_user_id, n.status, n.start_at, n.end_at, n.notes,
		       n.created_at, n.updated_at, n.deleted_at,
		       COALESCE(u.callsign, ''),
		       (SELECT COUNT(*) FROM checkins c WHERE c.net_id = n.id AND c.deleted_at IS NULL),
		       `+canManageExpr+`
		FROM nets n
		LEFT JOIN users u ON u.id = n.ncs_user_id
		WHERE n.deleted_at IS NULL
		ORDER BY n.end_at IS NULL DESC, n.end_at DESC, n.created_at DESC`, b2i(isAdmin), requesterID)
	if err != nil {
		return nil, fmt.Errorf("list nets: %w", err)
	}
	defer rows.Close()

	var out []models.NetWithMeta
	for rows.Next() {
		var n models.Net
		var ncs, start, end, deleted sql.NullString
		var canManage int
		var m models.NetWithMeta
		if err := rows.Scan(&n.ID, &n.Name, &n.NetDate, &ncs, &n.Status, &start, &end, &n.Notes,
			&n.CreatedAt, &n.UpdatedAt, &deleted, &n.NCSCallsign, &m.CheckInCount, &canManage); err != nil {
			return nil, err
		}
		n.NCSUserID = ncs.String
		n.StartAt = nullToPtr(start)
		n.EndAt = nullToPtr(end)
		n.DeletedAt = nullToPtr(deleted)
		n.CanManage = canManage != 0
		m.Net = n
		out = append(out, m)
	}
	return out, rows.Err()
}

// ChangedNetsSince returns nets whose updated_at is strictly greater than since,
// including tombstones (soft-deleted), for the sync-down protocol. The NCS
// callsign is joined in so clients can display it offline; canManage is computed
// for requesterID so the SPA can gate edit controls per net.
func (s *Store) ChangedNetsSince(ctx context.Context, since, requesterID string, isAdmin bool) ([]models.Net, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT n.id, n.name, n.net_date, n.ncs_user_id, n.status, n.start_at, n.end_at, n.notes,
		       n.created_at, n.updated_at, n.deleted_at, COALESCE(u.callsign, ''),
		       `+canManageExpr+`
		FROM nets n LEFT JOIN users u ON u.id = n.ncs_user_id
		WHERE n.updated_at > ? ORDER BY n.updated_at`, b2i(isAdmin), requesterID, since)
	if err != nil {
		return nil, fmt.Errorf("changed nets: %w", err)
	}
	defer rows.Close()

	var out []models.Net
	for rows.Next() {
		var n models.Net
		var ncs, start, end, deleted sql.NullString
		var canManage int
		if err := rows.Scan(&n.ID, &n.Name, &n.NetDate, &ncs, &n.Status, &start, &end, &n.Notes,
			&n.CreatedAt, &n.UpdatedAt, &deleted, &n.NCSCallsign, &canManage); err != nil {
			return nil, err
		}
		n.NCSUserID = ncs.String
		n.StartAt = nullToPtr(start)
		n.EndAt = nullToPtr(end)
		n.DeletedAt = nullToPtr(deleted)
		n.CanManage = canManage != 0
		out = append(out, n)
	}
	return out, rows.Err()
}

// AddNetController records userID as a controller of netID (idempotent).
func (s *Store) AddNetController(ctx context.Context, netID, userID, now string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO net_controllers (net_id, user_id, created_at) VALUES (?, ?, ?)`,
		netID, userID, now)
	if err != nil {
		return fmt.Errorf("add net controller: %w", err)
	}
	return nil
}

// IsNetController reports whether userID is in netID's controller set.
func (s *Store) IsNetController(ctx context.Context, netID, userID string) (bool, error) {
	var ok int
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM net_controllers WHERE net_id = ? AND user_id = ?)`,
		netID, userID).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("is net controller: %w", err)
	}
	return ok != 0, nil
}

// SetNetNCS updates the current/displayed NCS of a net. It is the only writer of
// ncs_user_id (UpdateNet deliberately leaves it untouched); reassignment also
// adds the new NCS to net_controllers via AddNetController.
func (s *Store) SetNetNCS(ctx context.Context, netID, userID, updatedAt string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE nets SET ncs_user_id = ?, updated_at = ? WHERE id = ?`, userID, updatedAt, netID)
	if err != nil {
		return fmt.Errorf("set net ncs: %w", err)
	}
	return nil
}

func ptrToNull(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}
