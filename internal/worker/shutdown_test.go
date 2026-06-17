// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package worker

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestDriverCancelReapsPi is the mechanism behind AC4 (graceful shutdown): when
// the context is cancelled mid-job, exec.CommandContext kills the pi subprocess,
// so Run returns promptly instead of hanging on a still-working pi — and pi is
// reaped (no orphan). cmd/worker turns SIGTERM into exactly this ctx cancel via
// signal.NotifyContext, mirroring the motor.
func TestDriverCancelReapsPi(t *testing.T) {
	d := &Driver{
		Bin: buildFakePI(t),
		Dir: t.TempDir(),
		// fail.jsonl has no agent_end; with FAKEPI_HANG the process stays alive
		// after emitting events, so the job is genuinely "in flight" at cancel.
		Env: append(fixtureEnv(t, "fail.jsonl"), "FAKEPI_HANG=1"),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var firstOnce sync.Once
	firstSeen := make(chan struct{})
	done := make(chan struct{})

	var res Result
	var runErr error
	go func() {
		res, runErr = d.Run(ctx, "long running job", func(Event) {
			firstOnce.Do(func() { close(firstSeen) })
		})
		close(done)
	}()

	// Wait until pi is mid-job, then signal shutdown.
	select {
	case <-firstSeen:
	case <-time.After(5 * time.Second):
		t.Fatal("driver never delivered an event; fixture/harness broken")
	}
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after ctx cancel — pi was not terminated (orphan)")
	}

	if res.SawAgentEnd {
		t.Error("SawAgentEnd = true, but the hang fixture never emits agent_end")
	}
	// A process killed by the context is not a clean (0) exit.
	if res.ExitCode == 0 {
		t.Errorf("ExitCode = 0 after cancel, want non-zero (killed); runErr=%v", runErr)
	}
}
