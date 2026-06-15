// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package db

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// migrationsFS embeds the numbered up/down files into the binary. The same
// schema therefore ships inside the distroless image — there are no .sql files
// on disk in production, and what runs is byte-identical to what was tested.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// newMigrator wires the embedded source to a golang-migrate instance over a
// fresh connection to path. The caller owns conn and must Close it.
func newMigrator(path string) (*migrate.Migrate, error) {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("db: load embedded migrations: %w", err)
	}

	conn, err := Open(path)
	if err != nil {
		return nil, err
	}

	driver, err := sqlite.WithInstance(conn, &sqlite.Config{})
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("db: migrate driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "sqlite", driver)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("db: migrate init: %w", err)
	}
	return m, nil
}

// Migrate applies all pending up migrations to the SQLite database at path.
// It runs on every motor boot and behind `motor migrate up` / `make migrate-up`
// — one code path, so dev and prod can never diverge. Idempotent: an
// already-current database returns nil (ErrNoChange is success, not failure).
func Migrate(path string) error {
	m, err := newMigrator(path)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("db: migrate up: %w", err)
	}
	return nil
}

// MigrateDown rolls the schema all the way back to empty. Destructive and
// dev-only — exposed via `motor migrate down` / `make migrate-down` to reset a
// local database, never wired into the server boot path.
func MigrateDown(path string) error {
	m, err := newMigrator(path)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("db: migrate down: %w", err)
	}
	return nil
}
