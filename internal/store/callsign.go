package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"netlog/internal/models"
)

const callsignColumns = `callsign, first_name, last_name, nickname, address1, address2, city,
	state, zip, country, dxcc, grid, latitude, longitude, cq_zone, itu_zone, iota, continent,
	email, website, qsl_manager, lotw, eqsl, flag_iso2, source, last_lookup_at`

// GetCallsign returns cached callbook/DXCC data for a callsign, or ErrNotFound.
func (s *Store) GetCallsign(ctx context.Context, callsign string) (models.CallsignData, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+callsignColumns+` FROM callsign_cache WHERE callsign = ?`, callsign)
	var d models.CallsignData
	var dxcc, cq, itu sql.NullInt64
	var lat, lon sql.NullFloat64
	var last sql.NullString
	err := row.Scan(&d.Callsign, &d.FirstName, &d.LastName, &d.Nickname, &d.Address1, &d.Address2,
		&d.City, &d.State, &d.Zip, &d.Country, &dxcc, &d.Grid, &lat, &lon, &cq, &itu, &d.IOTA,
		&d.Continent, &d.Email, &d.Website, &d.QSLManager, &d.LoTW, &d.EQSL, &d.FlagISO2, &d.Source, &last)
	if errors.Is(err, sql.ErrNoRows) {
		return models.CallsignData{}, ErrNotFound
	}
	if err != nil {
		return models.CallsignData{}, fmt.Errorf("get callsign: %w", err)
	}
	if dxcc.Valid {
		v := int(dxcc.Int64)
		d.DXCC = &v
	}
	if cq.Valid {
		v := int(cq.Int64)
		d.CQZone = &v
	}
	if itu.Valid {
		v := int(itu.Int64)
		d.ITUZone = &v
	}
	if lat.Valid {
		d.Latitude = &lat.Float64
	}
	if lon.Valid {
		d.Longitude = &lon.Float64
	}
	if last.Valid {
		d.LastLookupAt = &last.String
	}
	return d, nil
}

// UpsertCallsign inserts or updates cached data. Raw provider payloads are only
// overwritten when supplied (a nil raw preserves the other provider's stored
// payload), so a primary-only refresh never discards fallback data.
func (s *Store) UpsertCallsign(ctx context.Context, d models.CallsignData, rawQRZ, rawHamQTH *string) error {
	now := models.Now()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO callsign_cache (
			callsign, first_name, last_name, nickname, address1, address2, city, state, zip,
			country, dxcc, grid, latitude, longitude, cq_zone, itu_zone, iota, continent, email,
			website, qsl_manager, lotw, eqsl, flag_iso2, source, raw_qrz, raw_hamqth,
			last_lookup_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(callsign) DO UPDATE SET
			first_name=excluded.first_name, last_name=excluded.last_name, nickname=excluded.nickname,
			address1=excluded.address1, address2=excluded.address2, city=excluded.city,
			state=excluded.state, zip=excluded.zip, country=excluded.country, dxcc=excluded.dxcc,
			grid=excluded.grid, latitude=excluded.latitude, longitude=excluded.longitude,
			cq_zone=excluded.cq_zone, itu_zone=excluded.itu_zone, iota=excluded.iota,
			continent=excluded.continent, email=excluded.email, website=excluded.website,
			qsl_manager=excluded.qsl_manager, lotw=excluded.lotw, eqsl=excluded.eqsl,
			flag_iso2=excluded.flag_iso2, source=excluded.source,
			raw_qrz=COALESCE(excluded.raw_qrz, callsign_cache.raw_qrz),
			raw_hamqth=COALESCE(excluded.raw_hamqth, callsign_cache.raw_hamqth),
			last_lookup_at=excluded.last_lookup_at, updated_at=excluded.updated_at`,
		d.Callsign, d.FirstName, d.LastName, d.Nickname, d.Address1, d.Address2, d.City, d.State,
		d.Zip, d.Country, nullInt(d.DXCC), d.Grid, nullFloat(d.Latitude), nullFloat(d.Longitude),
		nullInt(d.CQZone), nullInt(d.ITUZone), d.IOTA, d.Continent, d.Email, d.Website,
		d.QSLManager, d.LoTW, d.EQSL, d.FlagISO2, d.Source, rawQRZ, rawHamQTH,
		nullStr(d.LastLookupAt), now, now)
	if err != nil {
		return fmt.Errorf("upsert callsign: %w", err)
	}
	return nil
}

func nullInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func nullFloat(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

func nullStr(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}
