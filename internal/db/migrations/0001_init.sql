-- +goose Up

-- Users. password_hash is NULL for OIDC-only accounts. The first account ever
-- created is forced to be an admin (enforced in application logic).
CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    callsign      TEXT NOT NULL UNIQUE,
    first_name    TEXT NOT NULL,
    last_name     TEXT NOT NULL,
    email         TEXT NOT NULL,
    password_hash TEXT,
    role          TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);

-- Linked OIDC identities. A user may link one identity per issuer.
CREATE TABLE user_oidc (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    issuer     TEXT NOT NULL,
    subject    TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (issuer, subject)
);

-- Server-side sessions. The cookie carries the opaque id only.
CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
);
CREATE INDEX idx_sessions_user ON sessions (user_id);
CREATE INDEX idx_sessions_expires ON sessions (expires_at);

-- Nets. status: pending (created, not started), open (live), closed (read-only).
-- start_at is set once when opened; end_at once when closed. Both are UTC.
-- updated_at + deleted_at drive the sync-down protocol (tombstones).
CREATE TABLE nets (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    net_date     TEXT NOT NULL,
    ncs_user_id  TEXT NOT NULL REFERENCES users (id),
    status       TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'open', 'closed')),
    start_at     TEXT,
    end_at       TEXT,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    deleted_at   TEXT
);
CREATE INDEX idx_nets_end_at ON nets (end_at);
CREATE INDEX idx_nets_updated_at ON nets (updated_at);

-- Check-ins logged against a net.
CREATE TABLE checkins (
    id            TEXT PRIMARY KEY,
    net_id        TEXT NOT NULL REFERENCES nets (id) ON DELETE CASCADE,
    callsign      TEXT NOT NULL,
    nickname      TEXT NOT NULL DEFAULT '',
    has_traffic   INTEGER NOT NULL DEFAULT 0 CHECK (has_traffic IN (0, 1)),
    short_time    INTEGER NOT NULL DEFAULT 0 CHECK (short_time IN (0, 1)),
    notes         TEXT NOT NULL DEFAULT '',
    seq           INTEGER NOT NULL DEFAULT 0,
    checked_in_at TEXT NOT NULL,
    created_by    TEXT REFERENCES users (id),
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    deleted_at    TEXT
);
CREATE INDEX idx_checkins_net ON checkins (net_id);
CREATE INDEX idx_checkins_callsign ON checkins (callsign);
CREATE INDEX idx_checkins_updated_at ON checkins (updated_at);

-- Cached callbook + DXCC data per callsign. raw_qrz / raw_hamqth hold the full
-- normalized provider payloads (JSON) so we never lose data we didn't model.
-- The normalized columns may be extended by a forward migration after the
-- live-API field-coverage review.
CREATE TABLE callsign_cache (
    callsign       TEXT PRIMARY KEY,
    first_name     TEXT NOT NULL DEFAULT '',
    last_name      TEXT NOT NULL DEFAULT '',
    nickname       TEXT NOT NULL DEFAULT '',
    address1       TEXT NOT NULL DEFAULT '',
    address2       TEXT NOT NULL DEFAULT '',
    city           TEXT NOT NULL DEFAULT '',
    state          TEXT NOT NULL DEFAULT '',
    zip            TEXT NOT NULL DEFAULT '',
    country        TEXT NOT NULL DEFAULT '',
    dxcc           INTEGER,
    grid           TEXT NOT NULL DEFAULT '',
    latitude       REAL,
    longitude      REAL,
    cq_zone        INTEGER,
    itu_zone       INTEGER,
    iota           TEXT NOT NULL DEFAULT '',
    continent      TEXT NOT NULL DEFAULT '',
    email          TEXT NOT NULL DEFAULT '',
    website        TEXT NOT NULL DEFAULT '',
    qsl_manager    TEXT NOT NULL DEFAULT '',
    lotw           TEXT NOT NULL DEFAULT '',
    eqsl           TEXT NOT NULL DEFAULT '',
    flag_iso2      TEXT NOT NULL DEFAULT '',
    source         TEXT NOT NULL DEFAULT '',
    raw_qrz        TEXT,
    raw_hamqth     TEXT,
    last_lookup_at TEXT,
    created_at     TEXT NOT NULL,
    updated_at     TEXT NOT NULL
);

-- Single-row metadata about the cached cty.xml DXCC dataset.
CREATE TABLE cty_meta (
    id                 INTEGER PRIMARY KEY CHECK (id = 1),
    last_downloaded_at TEXT,
    source_date        TEXT,
    entity_count       INTEGER NOT NULL DEFAULT 0
);
INSERT INTO cty_meta (id, entity_count) VALUES (1, 0);

-- +goose Down
DROP TABLE cty_meta;
DROP TABLE callsign_cache;
DROP TABLE checkins;
DROP TABLE nets;
DROP TABLE sessions;
DROP TABLE user_oidc;
DROP TABLE users;
