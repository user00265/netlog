package store

import (
	"context"
	"database/sql"
	"fmt"
)

// CtyMeta describes the cached cty.xml DXCC dataset.
type CtyMeta struct {
	LastDownloadedAt string
	SourceDate       string
	EntityCount      int
}

// GetCtyMeta returns the cty dataset metadata (single row id=1).
func (s *Store) GetCtyMeta(ctx context.Context) (CtyMeta, error) {
	var m CtyMeta
	var last, src sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT last_downloaded_at, source_date, entity_count FROM cty_meta WHERE id = 1`).
		Scan(&last, &src, &m.EntityCount)
	if err != nil {
		return CtyMeta{}, fmt.Errorf("get cty meta: %w", err)
	}
	m.LastDownloadedAt = last.String
	m.SourceDate = src.String
	return m, nil
}

// SetCtyMeta records a successful cty dataset refresh.
func (s *Store) SetCtyMeta(ctx context.Context, downloadedAt, sourceDate string, entityCount int) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE cty_meta SET last_downloaded_at = ?, source_date = ?, entity_count = ? WHERE id = 1`,
		downloadedAt, sourceDate, entityCount)
	if err != nil {
		return fmt.Errorf("set cty meta: %w", err)
	}
	return nil
}
