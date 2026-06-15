-- CreateTeam inserts a new team (the multi-tenant root) and returns the row.
-- name: CreateTeam :one
INSERT INTO teams (id, name)
VALUES (?, ?)
RETURNING id, name, created_at, updated_at;

-- GetTeam returns the team with the given id, or sql.ErrNoRows if absent.
-- name: GetTeam :one
SELECT id, name, created_at, updated_at FROM teams
WHERE id = ?;
