-- +goose Up
-- net_controllers is the full (growing) set of users who have held NCS control
-- of a net. nets.ncs_user_id remains the *current/displayed* NCS; this table is
-- what edit authorization checks, so handing a net off grants the new NCS access
-- while every prior NCS retains it.
CREATE TABLE IF NOT EXISTS net_controllers (
  net_id     TEXT NOT NULL REFERENCES nets(id) ON DELETE CASCADE,
  user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TEXT NOT NULL,
  PRIMARY KEY (net_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_net_controllers_user ON net_controllers(user_id);

-- Backfill: each existing net's current NCS becomes its first controller.
INSERT OR IGNORE INTO net_controllers (net_id, user_id, created_at)
  SELECT id, ncs_user_id, created_at FROM nets
  WHERE ncs_user_id IS NOT NULL AND ncs_user_id <> '';

-- +goose Down
DROP TABLE net_controllers;
