// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package db_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/mlorentedev/iris/internal/db"
	"github.com/mlorentedev/iris/internal/db/sqlc"
)

// newTestQueries migrates a throwaway SQLite file (auto-removed with the test's
// temp dir) and returns a Queries bound to it.
func newTestQueries(t *testing.T) *sqlc.Queries {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	if err := db.Migrate(path); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	conn, err := db.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return sqlc.New(conn)
}

// TestMigrateIsIdempotent asserts the startup auto-migrate is safe to run on an
// already-current database — the property the server boot path relies on.
func TestMigrateIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	if err := db.Migrate(path); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := db.Migrate(path); err != nil {
		t.Fatalf("second migrate must be a no-op, got: %v", err)
	}
}

// TestTeamsRoundTrip exercises the full chain: migration schema -> sqlc types ->
// runtime insert/select, and confirms the audit columns default-populate.
func TestTeamsRoundTrip(t *testing.T) {
	q := newTestQueries(t)
	ctx := context.Background()

	created, err := q.CreateTeam(ctx, sqlc.CreateTeamParams{ID: "acme", Name: "Acme Corp"})
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if created.ID != "acme" || created.Name != "Acme Corp" {
		t.Errorf("created = %+v, want id=acme name=Acme Corp", created)
	}
	if created.CreatedAt == "" || created.UpdatedAt == "" {
		t.Errorf("audit columns not default-populated: %+v", created)
	}

	got, err := q.GetTeam(ctx, "acme")
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	if got != created {
		t.Errorf("GetTeam = %+v, want %+v", got, created)
	}
}

// TestTeamsRejectBlankID proves the multi-tenant discriminator invariant is
// enforced by the database itself (CHECK + STRICT), not merely by convention —
// a blank team_id would be a cross-tenant data-leak hazard.
func TestTeamsRejectBlankID(t *testing.T) {
	q := newTestQueries(t)
	if _, err := q.CreateTeam(context.Background(), sqlc.CreateTeamParams{ID: "  ", Name: "x"}); err == nil {
		t.Fatal("blank team id must violate the CHECK constraint, got nil error")
	}
}
