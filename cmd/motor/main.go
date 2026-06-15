// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

// Command motor is the iris control plane: HTTP API + scheduler + runtime adapter
// + persistence, as a single binary (per ADR-002). This skeleton serves the health
// endpoints; subsystems land in later SDD-034 tasks.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sethvargo/go-envconfig"

	"github.com/mlorentedev/iris/internal/api"
	"github.com/mlorentedev/iris/internal/db"
)

// config is the motor's 12-factor environment configuration.
type config struct {
	Addr            string        `env:"IRIS_ADDR, default=:8080"`
	ShutdownTimeout time.Duration `env:"IRIS_SHUTDOWN_TIMEOUT, default=15s"`
	DBPath          string        `env:"IRIS_DB_PATH, default=iris.db"`
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("motor exited with error", "err", err)
		os.Exit(1)
	}
}

// run wires config, the HTTP server, and graceful shutdown on SIGINT/SIGTERM.
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var cfg config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		return err
	}

	// Subcommand: `motor migrate up|down`. Applies schema and exits, sharing the
	// same embedded migrations and DB path as the server boot path so dev
	// (`make migrate-up`) and prod (auto-migrate below) can never drift.
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		return runMigrate(cfg.DBPath, os.Args[2:])
	}

	// Server path: apply pending migrations on boot, then open the connection
	// the HTTP layer shares for its readiness probe.
	if err := db.Migrate(cfg.DBPath); err != nil {
		return err
	}
	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.NewRouter(conn),
		ReadHeaderTimeout: 5 * time.Second, // gosec G112: bound slow-header clients
	}

	srvErr := make(chan error, 1)
	go func() {
		slog.Info("motor listening", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErr <- err
		}
	}()

	select {
	case err := <-srvErr:
		return err
	case <-ctx.Done():
		slog.Info("shutdown signal received, draining", "timeout", cfg.ShutdownTimeout)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		slog.Info("motor stopped cleanly")
		return nil
	}
}

// runMigrate handles `motor migrate up|down`. Direction defaults to up.
func runMigrate(dbPath string, args []string) error {
	dir := "up"
	if len(args) > 0 {
		dir = args[0]
	}

	switch dir {
	case "up":
		if err := db.Migrate(dbPath); err != nil {
			return err
		}
		slog.Info("migrations applied", "db", dbPath)
	case "down":
		if err := db.MigrateDown(dbPath); err != nil {
			return err
		}
		slog.Info("migrations rolled back", "db", dbPath)
	default:
		return fmt.Errorf("motor migrate: unknown direction %q (want: up | down)", dir)
	}
	return nil
}
