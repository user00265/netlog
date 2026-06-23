-- +goose Up
-- Operator display preferences (IANA timezone + 12h/24h clock).
ALTER TABLE users ADD COLUMN timezone TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN time_format TEXT NOT NULL DEFAULT '24h';
-- Free-form NCS notes captured during a net.
ALTER TABLE nets ADD COLUMN notes TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE nets DROP COLUMN notes;
ALTER TABLE users DROP COLUMN time_format;
ALTER TABLE users DROP COLUMN timezone;
