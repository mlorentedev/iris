// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

// Command worker is the iris fleet worker (ADR-009): a thin Go supervisor that
// subscribes to its NATS subject, drives a pi coding session in an isolated git
// worktree per job, and streams typed telemetry back on the team activity
// channel. One job at a time — fleet concurrency is more workers, not more jobs
// per worker (ADR-002 isolation).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sethvargo/go-envconfig"

	"github.com/mlorentedev/iris/internal/protocol"
	"github.com/mlorentedev/iris/internal/worker"
)

// Build metadata, injected at link time via -ldflags (same scheme as the motor).
var (
	version   = "0.0.0-dev"
	gitSHA    = "unknown"
	buildDate = "unknown"
)

func versionString() string {
	return fmt.Sprintf("iris-worker %s (sha=%s built=%s)", version, gitSHA, buildDate)
}

// config is the worker's 12-factor environment configuration. TEAM and
// WORKER_NAME are required: a worker has no identity without them.
type config struct {
	NATSURL         string        `env:"NATS_URL, default=nats://nats:4222"`
	Team            string        `env:"TEAM, required"`
	Name            string        `env:"WORKER_NAME, required"`
	Repo            string        `env:"WORKER_REPO, default=/workspace"`
	WorktreeBase    string        `env:"WORKER_WORKTREE_BASE, default=/tmp/iris-worktrees"`
	PiBin           string        `env:"PI_BIN, default=pi"`
	Model           string        `env:"PI_MODEL"`
	ShutdownTimeout time.Duration `env:"IRIS_SHUTDOWN_TIMEOUT, default=15s"`
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("worker exited with error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	// `worker --version` prints the build triplet and exits — dependency-free so
	// it works in the image without NATS or env (parity with the motor).
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Println(versionString())
		return nil
	}

	// SIGINT/SIGTERM cancel ctx; the cancellation propagates to the in-flight pi
	// subprocess (exec.CommandContext) and breaks the pull loop — graceful
	// shutdown with no orphaned pi (AC4).
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var cfg config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		return err
	}

	subject, err := protocol.SubjectFor(cfg.Team, protocol.SubjectWorker, cfg.Name)
	if err != nil {
		return fmt.Errorf("worker: invalid team/name: %w", err)
	}

	bus, err := worker.ConnectNATS(cfg.NATSURL, "iris-worker-"+cfg.Name, cfg.ShutdownTimeout)
	if err != nil {
		return err
	}
	defer func() { _ = bus.Drain() }()

	w := &worker.Worker{
		Team:  cfg.Team,
		Name:  cfg.Name,
		Repo:  cfg.Repo,
		Base:  cfg.WorktreeBase,
		PiBin: cfg.PiBin,
		Model: cfg.Model,
		Env:   os.Environ(), // inference provider config rides through to pi
		Pub:   bus,
	}
	w.EmitStartup(ctx)

	sub, err := bus.SubscribeSync(subject)
	if err != nil {
		return err
	}
	slog.Info("worker ready", "team", cfg.Team, "name", cfg.Name, "subject", subject, "nats", cfg.NATSURL)

	// Single-job pull loop: one message handled to completion before the next.
	for {
		msg, err := sub.NextMsgWithContext(ctx)
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("shutdown signal received, draining", "timeout", cfg.ShutdownTimeout)
				return nil
			}
			slog.Error("worker: receive message", "err", err, "subject", subject)
			continue
		}
		if err := w.Handle(ctx, msg.Data); err != nil {
			// The failure is already surfaced as an activity_event inside Handle;
			// log it here too. A failed job never kills the worker.
			if errors.Is(err, context.Canceled) {
				return nil // shutdown mid-job; pi already reaped
			}
			slog.Error("worker: job failed", "err", err)
		}
	}
}
