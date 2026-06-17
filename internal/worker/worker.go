// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package worker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mlorentedev/iris/internal/protocol"
)

// ActivityJobFailed is the activity_event "event" value the worker emits when a
// job fails. It is an iris-level event (not part of pi's vocabulary): the point
// is that a failure is published, never silently dropped (proposal item B).
const ActivityJobFailed = "job_failed"

// Publisher is the narrow seam the worker uses to emit telemetry. The real
// implementation wraps a NATS connection; tests use a fake. Keeping the worker's
// orchestration off the concrete NATS client is what makes the job lifecycle
// unit-testable without a broker.
type Publisher interface {
	Publish(subject string, data []byte) error
}

// Worker orchestrates one job at a time (ADR-002 isolation; fleet concurrency is
// more workers, not more jobs per worker). It owns the worker identity, the repo
// that worktrees branch from, and the pi configuration; Handle drives a single
// job end to end.
type Worker struct {
	Team  string    // team this worker belongs to (NATS subject tenant)
	Name  string    // this worker's name (subject leaf + envelope From)
	Repo  string    // repository git worktrees are added from
	Base  string    // base dir for per-job worktrees
	PiBin string    // pi binary; default "pi"
	Model string    // optional pi --model passthrough
	Env   []string  // process env (inference passthrough); nil inherits
	Pub   Publisher // telemetry sink
}

// Handle runs one job: decode the envelope, create an isolated worktree, drive pi
// in it while streaming activity, and clean the worktree up — on every exit path.
// A failed job is BOTH returned as an error and surfaced as a failure
// activity_event, so callers can react and the UI never loses a failure.
func (w *Worker) Handle(ctx context.Context, raw []byte) error {
	job, err := DecodeJob(raw)
	if err != nil {
		// Undecodable: we don't know the team to address activity to, so the
		// best we can do is return the error to the subscription loop to log.
		return fmt.Errorf("worker: decode job: %w", err)
	}

	wt, err := NewWorktree(ctx, w.Repo, w.Base, job.MessageID)
	if err != nil {
		w.publishFailure(ctx, job, fmt.Sprintf("worktree setup failed: %v", err))
		return err
	}
	defer func() {
		if rmErr := wt.Remove(ctx); rmErr != nil {
			slog.Error("worker: worktree cleanup failed (leaked)", "err", rmErr, "job", job.MessageID)
		}
	}()

	res, runErr := w.driverFor(wt.Dir).Run(ctx, job.Prompt, func(e Event) {
		if a, ok := ActivityFor(e); ok {
			w.publishActivity(ctx, job, a)
		}
	})
	if runErr != nil {
		w.publishFailure(ctx, job, fmt.Sprintf("pi stream error: %v", runErr))
		return runErr
	}
	// Success contract: pi exited cleanly AND completed its turn (agent_end). An
	// exit 0 without agent_end is a truncated run, treated as failure.
	if res.ExitCode != 0 || !res.SawAgentEnd {
		summary := fmt.Sprintf("pi exited %d (agent_end=%v)", res.ExitCode, res.SawAgentEnd)
		w.publishFailure(ctx, job, summary)
		return fmt.Errorf("worker: job %s failed: %s", job.MessageID, summary)
	}
	return nil
}

// EmitStartup publishes a container_validation envelope announcing readiness:
// pi resolvable on PATH and the worktree base writable. It runs once at boot.
func (w *Worker) EmitStartup(ctx context.Context) {
	var checks []string
	var errs []string

	bin := w.piBin()
	if _, err := exec.LookPath(bin); err != nil {
		errs = append(errs, fmt.Sprintf("pi binary %q not found on PATH: %v", bin, err))
	} else {
		checks = append(checks, "pi_present")
	}

	if err := baseWritable(w.Base); err != nil {
		errs = append(errs, fmt.Sprintf("worktree base %q not writable: %v", w.Base, err))
	} else {
		checks = append(checks, "worktree_base_writable")
	}

	status := "ok"
	if len(errs) > 0 {
		status = "error"
	}
	w.publish(ctx, protocol.TypeContainerValidation, protocol.Message{}, protocol.ContainerValidationPayload{
		Status: status,
		Checks: checks,
		Errors: errs,
	})
}

// driverFor builds a per-job pi driver bound to the worktree dir, so no shared
// driver state (e.g. the args slice) leaks across concurrent jobs in a fleet.
func (w *Worker) driverFor(dir string) *Driver {
	var args []string
	if w.Model != "" {
		args = []string{"--model", w.Model}
	}
	return &Driver{Bin: w.piBin(), Args: args, Dir: dir, Env: w.Env}
}

func (w *Worker) piBin() string {
	if w.PiBin == "" {
		return "pi"
	}
	return w.PiBin
}

func (w *Worker) publishActivity(ctx context.Context, job Job, payload protocol.ActivityEventPayload) {
	w.publish(ctx, protocol.TypeActivityEvent, job.Envelope, payload)
}

func (w *Worker) publishFailure(ctx context.Context, job Job, summary string) {
	w.publish(ctx, protocol.TypeActivityEvent, job.Envelope, protocol.ActivityEventPayload{
		Event:   ActivityJobFailed,
		Summary: summary,
	})
}

// publish builds an envelope, stamps team/threading from the originating job (so
// the UI can group activity by team and conversation), and sends it to the team
// activity subject. Failures to publish are logged, not propagated: telemetry
// must not crash a job. The `origin` envelope carries threading context; pass a
// zero Message when there is none (startup).
func (w *Worker) publish(_ context.Context, mtype protocol.MessageType, origin protocol.Message, payload any) {
	msg, err := protocol.NewMessage(w.Name, "", mtype, payload)
	if err != nil {
		slog.Error("worker: build envelope", "err", err, "type", mtype)
		return
	}
	msg.Context.TeamID = w.Team
	msg.Context.ThreadID = origin.Context.ThreadID
	msg.Context.TraceID = origin.Context.TraceID
	if origin.MessageID != "" {
		msg.RefMessageID = origin.MessageID
	}

	subject, err := protocol.SubjectFor(w.Team, protocol.SubjectActivity)
	if err != nil {
		slog.Error("worker: activity subject", "err", err, "team", w.Team)
		return
	}
	data, err := msg.Marshal()
	if err != nil {
		slog.Error("worker: marshal envelope", "err", err, "type", mtype)
		return
	}
	if err := w.Pub.Publish(subject, data); err != nil {
		slog.Error("worker: publish", "err", err, "subject", subject, "type", mtype)
	}
}

// baseWritable confirms dir exists (creating it if needed) and accepts a write.
func baseWritable(dir string) error {
	if dir == "" {
		return fmt.Errorf("base dir is empty")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	probe := filepath.Join(dir, ".iris-write-probe")
	if err := os.WriteFile(probe, []byte("ok"), 0o600); err != nil {
		return err
	}
	return os.Remove(probe)
}
