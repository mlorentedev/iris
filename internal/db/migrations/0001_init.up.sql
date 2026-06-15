-- SPDX-License-Identifier: Apache-2.0
-- Copyright 2026 Manu Lorente
--
-- 0001_init: the teams table, iris's multi-tenant root.
--
-- `id` IS the team_id discriminator (FROZEN per bootstrap-contract section 3):
-- every other persisted model carries a team_id FK back to this row, and
-- middleware injects it on every request (ADR-002 multi-tenant boundary). It is
-- TEXT, not an autoincrement INTEGER, so a team's identity is a stable, URL-safe
-- slug that never collides across environments (dev/kubelab/customer): a
-- precondition for the smoke-test being reproducible on a clean machine.
--
-- Audit columns are stored as ISO-8601 TEXT (SQLite has no native datetime type;
-- TEXT keeps round-trips driver-agnostic between modernc/sqlite and any future
-- Postgres adapter). DEFAULTs let callers insert (id, name) alone.
--
-- Keep this file ASCII-only (sqlc corrupts SQL constants around multi-byte chars).

CREATE TABLE teams (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),

    CONSTRAINT teams_id_not_blank   CHECK (length(trim(id)) > 0),
    CONSTRAINT teams_name_not_blank CHECK (length(trim(name)) > 0)
) STRICT;

-- Keep updated_at honest without forcing every caller to set it.
CREATE TRIGGER teams_set_updated_at
AFTER UPDATE ON teams
FOR EACH ROW
BEGIN
    UPDATE teams
    SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
    WHERE id = NEW.id;
END;
