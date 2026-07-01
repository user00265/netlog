-- +goose NO TRANSACTION
-- A net is now unassigned (no NCS) until someone opens it, so ncs_user_id must
-- allow NULL. SQLite can't drop NOT NULL in place; rebuild the table following
-- the documented 12-step procedure. foreign_keys is toggled off so dropping the
-- old nets table doesn't trip net_controllers' FK, and so the rebuild can't run
-- inside a transaction (hence NO TRANSACTION).

-- +goose Up
PRAGMA foreign_keys=OFF;
CREATE TABLE nets_new (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    net_date     TEXT NOT NULL,
    ncs_user_id  TEXT REFERENCES users (id),
    status       TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'open', 'closed')),
    start_at     TEXT,
    end_at       TEXT,
    notes        TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    deleted_at   TEXT
);
INSERT INTO nets_new (id, name, net_date, ncs_user_id, status, start_at, end_at, notes, created_at, updated_at, deleted_at)
  SELECT id, name, net_date, ncs_user_id, status, start_at, end_at, notes, created_at, updated_at, deleted_at FROM nets;
DROP TABLE nets;
ALTER TABLE nets_new RENAME TO nets;
CREATE INDEX idx_nets_end_at ON nets (end_at);
CREATE INDEX idx_nets_updated_at ON nets (updated_at);
PRAGMA foreign_keys=ON;

-- +goose Down
PRAGMA foreign_keys=OFF;
CREATE TABLE nets_old (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    net_date     TEXT NOT NULL,
    ncs_user_id  TEXT NOT NULL REFERENCES users (id),
    status       TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'open', 'closed')),
    start_at     TEXT,
    end_at       TEXT,
    notes        TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    deleted_at   TEXT
);
INSERT INTO nets_old (id, name, net_date, ncs_user_id, status, start_at, end_at, notes, created_at, updated_at, deleted_at)
  SELECT id, name, net_date, ncs_user_id, status, start_at, end_at, notes, created_at, updated_at, deleted_at FROM nets WHERE ncs_user_id IS NOT NULL;
DROP TABLE nets;
ALTER TABLE nets_old RENAME TO nets;
CREATE INDEX idx_nets_end_at ON nets (end_at);
CREATE INDEX idx_nets_updated_at ON nets (updated_at);
PRAGMA foreign_keys=ON;
