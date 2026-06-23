package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"netlog/internal/models"
)

const checkinColumns = `id, net_id, callsign, nickname, has_traffic, short_time, notes, seq,
	checked_in_at, created_by, created_at, updated_at, deleted_at`

func scanCheckin(row interface{ Scan(...any) error }) (models.CheckIn, error) {
	var c models.CheckIn
	var createdBy, deleted sql.NullString
	var hasTraffic, shortTime int
	err := row.Scan(&c.ID, &c.NetID, &c.Callsign, &c.Nickname, &hasTraffic, &shortTime, &c.Notes,
		&c.Seq, &c.CheckedInAt, &createdBy, &c.CreatedAt, &c.UpdatedAt, &deleted)
	if err != nil {
		return models.CheckIn{}, err
	}
	c.HasTraffic = hasTraffic == 1
	c.ShortTime = shortTime == 1
	c.CreatedBy = nullToPtr(createdBy)
	c.DeletedAt = nullToPtr(deleted)
	return c, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// CreateCheckin inserts a new check-in.
func (s *Store) CreateCheckin(ctx context.Context, c models.CheckIn) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO checkins (`+checkinColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.NetID, c.Callsign, c.Nickname, boolToInt(c.HasTraffic), boolToInt(c.ShortTime),
		c.Notes, c.Seq, c.CheckedInAt, ptrToNull(c.CreatedBy), c.CreatedAt, c.UpdatedAt, ptrToNull(c.DeletedAt))
	if err != nil {
		return fmt.Errorf("create checkin: %w", err)
	}
	return nil
}

// UpdateCheckin writes all mutable fields of an existing check-in.
func (s *Store) UpdateCheckin(ctx context.Context, c models.CheckIn) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE checkins SET callsign=?, nickname=?, has_traffic=?, short_time=?, notes=?, seq=?,
		 checked_in_at=?, updated_at=?, deleted_at=? WHERE id=?`,
		c.Callsign, c.Nickname, boolToInt(c.HasTraffic), boolToInt(c.ShortTime), c.Notes, c.Seq,
		c.CheckedInAt, c.UpdatedAt, ptrToNull(c.DeletedAt), c.ID)
	if err != nil {
		return fmt.Errorf("update checkin: %w", err)
	}
	return nil
}

// GetCheckin returns a check-in by id (including soft-deleted), or ErrNotFound.
func (s *Store) GetCheckin(ctx context.Context, id string) (models.CheckIn, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+checkinColumns+` FROM checkins WHERE id = ?`, id)
	c, err := scanCheckin(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.CheckIn{}, ErrNotFound
	}
	return c, err
}

// ListCheckins returns non-deleted check-ins for a net, ordered by sequence.
func (s *Store) ListCheckins(ctx context.Context, netID string) ([]models.CheckIn, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+checkinColumns+` FROM checkins WHERE net_id = ? AND deleted_at IS NULL ORDER BY seq, created_at`, netID)
	if err != nil {
		return nil, fmt.Errorf("list checkins: %w", err)
	}
	defer rows.Close()

	var out []models.CheckIn
	for rows.Next() {
		c, err := scanCheckin(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// NextCheckinSeq returns the next sequence number for a net (max+1).
func (s *Store) NextCheckinSeq(ctx context.Context, netID string) (int, error) {
	var maxSeq sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT MAX(seq) FROM checkins WHERE net_id = ?`, netID).Scan(&maxSeq)
	if err != nil {
		return 0, fmt.Errorf("next checkin seq: %w", err)
	}
	return int(maxSeq.Int64) + 1, nil
}

// ChangedCheckinsSince returns check-ins changed after since, including
// tombstones, for the sync-down protocol.
func (s *Store) ChangedCheckinsSince(ctx context.Context, since string) ([]models.CheckIn, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+checkinColumns+` FROM checkins WHERE updated_at > ? ORDER BY updated_at`, since)
	if err != nil {
		return nil, fmt.Errorf("changed checkins: %w", err)
	}
	defer rows.Close()

	var out []models.CheckIn
	for rows.Next() {
		c, err := scanCheckin(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
