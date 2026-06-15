// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

// Package db owns iris's persistence: the sqlc-generated typed queries
// (internal/db/sqlc), the golang-migrate schema (internal/db/migrations), and
// the connection wiring. v0 is SQLite via the pure-Go modernc driver, which
// keeps the motor CGO_ENABLED=0 so it compiles into the distroless/static image
// (per bootstrap-contract § 3 — no GORM, no AutoMigrate).
package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // registers the pure-Go "sqlite" database/sql driver
)

// Open returns a *sql.DB for the SQLite file at path, configured with the
// pragmas iris depends on:
//
//   - journal_mode(WAL): readers don't block the single writer.
//   - foreign_keys(ON): SQLite leaves FK enforcement OFF by default; iris's
//     team_id discriminator is only safe if FKs are actually enforced.
//   - busy_timeout(5000): a contended writer retries for 5s instead of failing
//     immediately with "database is locked".
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)",
		path,
	)
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open %q: %w", path, err)
	}

	// A single SQLite file tolerates many concurrent readers but only one
	// writer. Capping open connections at 1 trades write throughput (not a
	// concern for a control-plane motor) for the elimination of lock contention
	// — the simplest correct default for v0.
	conn.SetMaxOpenConns(1)

	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("db: ping %q: %w", path, err)
	}
	return conn, nil
}
