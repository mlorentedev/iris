// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

// Package runtimetest provides a reusable conformance suite for AgentRuntime
// implementations. Every substrate adapter (Docker, Kubernetes, local-process)
// calls RunConformance from its own test to prove it honors the FROZEN base
// contract, so the lifecycle invariants live in one place instead of being
// re-derived per substrate. It is the AgentRuntime analog of the protocol
// package's committed wire fixtures: a shared oracle that keeps every
// implementation from drifting.
//
// The package imports testing in non-test code on purpose — the same idiom as
// net/http/httptest and testing/fstest — so the suite is importable by any
// substrate's _test.go.
package runtimetest

import (
	"context"
	"errors"
	"testing"

	"github.com/mlorentedev/iris/internal/runtime"
)

// teamConf is the team every conformance agent belongs to. Each subtest uses a
// distinct agent name so that substrates backed by shared state (a Docker daemon,
// a K8s cluster) do not collide across subtests.
const teamConf = "team-conf"

// Options parameterise the suite for the substrate under test. The base
// lifecycle is universal; what a "valid spec" or a "command that exits 0" looks
// like is substrate-specific, so the caller supplies those.
type Options struct {
	// NewSpec returns a valid, startable spec for the substrate, keyed by
	// identity. The suite copies it and mutates Image to synthesise drift, so
	// Image must be a meaningful field for the substrate (it is for Docker/K8s;
	// the local substrate may ignore it but must still hash it into SpecHash).
	NewSpec func(name, teamID string) runtime.AgentSpec

	// ExecProbe is a command known to exit 0 inside a started agent (e.g.
	// []string{"true"}). If nil, the suite skips the Exec invariant.
	ExecProbe []string
}

// RunConformance drives the base AgentRuntime contract against the runtime
// produced by newRuntime. newRuntime is called once per subtest so cases do not
// share an adapter instance; each agent a subtest starts is torn down via
// t.Cleanup so the suite leaves no residue on shared substrates. A failure names
// the precise invariant that broke.
func RunConformance(t *testing.T, newRuntime func() runtime.AgentRuntime, opts Options) {
	t.Helper()
	if newRuntime == nil {
		t.Fatal("runtimetest: newRuntime must not be nil")
	}
	if opts.NewSpec == nil {
		t.Fatal("runtimetest: Options.NewSpec must not be nil")
	}

	// startAgent starts a healthy agent and registers its teardown. Use it for
	// cases that need a running agent; cases that assert on Start itself call
	// Start directly.
	startAgent := func(t *testing.T, rt runtime.AgentRuntime, name string) (runtime.AgentHandle, runtime.AgentSpec) {
		t.Helper()
		spec := opts.NewSpec(name, teamConf)
		h, err := rt.Start(context.Background(), spec)
		if err != nil {
			t.Fatalf("Start(%s): %v", name, err)
		}
		t.Cleanup(func() { _ = rt.Stop(context.Background(), h) })
		return h, spec
	}

	t.Run("Start returns a populated handle", func(t *testing.T) {
		rt := newRuntime()
		h, spec := startAgent(t, rt, "agent-handle")
		if h.ID == "" {
			t.Error("handle.ID is empty; substrate must assign an identifier")
		}
		if h.Name != spec.Name {
			t.Errorf("handle.Name = %q, want %q", h.Name, spec.Name)
		}
		if h.TeamID != spec.TeamID {
			t.Errorf("handle.TeamID = %q, want %q", h.TeamID, spec.TeamID)
		}
		if h.Substrate == "" {
			t.Error("handle.Substrate is empty; substrate must stamp its type")
		}
	})

	t.Run("Status after Start is starting or running", func(t *testing.T) {
		rt := newRuntime()
		h, _ := startAgent(t, rt, "agent-status")
		st, err := rt.Status(context.Background(), h)
		if err != nil {
			t.Fatalf("Status: %v", err)
		}
		if st != runtime.StatusStarting && st != runtime.StatusRunning {
			t.Errorf("Status = %q, want starting or running", st)
		}
	})

	t.Run("Start is idempotent for an identical spec", func(t *testing.T) {
		rt := newRuntime()
		h1, spec := startAgent(t, rt, "agent-idem")
		h2, err := rt.Start(context.Background(), spec)
		if err != nil {
			t.Fatalf("second Start (identical spec) must not error, got: %v", err)
		}
		if h1.ID != h2.ID {
			t.Errorf("idempotent Start returned a new agent: %q != %q", h1.ID, h2.ID)
		}
	})

	t.Run("Start on spec drift returns ErrSpecDrift", func(t *testing.T) {
		rt := newRuntime()
		_, spec := startAgent(t, rt, "agent-drift")
		drift := spec
		drift.Image = spec.Image + "-drifted"
		if drift.SpecHash() == spec.SpecHash() {
			t.Fatal("test setup: drifted spec must hash differently; check SpecHash covers Image")
		}
		_, err := rt.Start(context.Background(), drift)
		if !errors.Is(err, runtime.ErrSpecDrift) {
			t.Errorf("Start on drift = %v, want ErrSpecDrift", err)
		}
	})

	t.Run("Stop is idempotent and unknown handles are no-ops", func(t *testing.T) {
		rt := newRuntime()
		h, _ := startAgent(t, rt, "agent-stop")
		if err := rt.Stop(context.Background(), h); err != nil {
			t.Fatalf("first Stop: %v", err)
		}
		if err := rt.Stop(context.Background(), h); err != nil {
			t.Errorf("second Stop must be a no-op, got: %v", err)
		}
		unknown := runtime.AgentHandle{ID: "does-not-exist", Name: "agent-ghost", TeamID: teamConf}
		if err := rt.Stop(context.Background(), unknown); err != nil {
			t.Errorf("Stop on unknown handle must be a no-op, got: %v", err)
		}
	})

	t.Run("Status on unknown handle returns ErrAgentNotFound", func(t *testing.T) {
		rt := newRuntime()
		unknown := runtime.AgentHandle{ID: "does-not-exist", Name: "agent-ghost", TeamID: teamConf}
		_, err := rt.Status(context.Background(), unknown)
		if !errors.Is(err, runtime.ErrAgentNotFound) {
			t.Errorf("Status on unknown handle = %v, want ErrAgentNotFound", err)
		}
	})

	t.Run("Status after Stop is stopped or not-found", func(t *testing.T) {
		rt := newRuntime()
		h, _ := startAgent(t, rt, "agent-poststop")
		if err := rt.Stop(context.Background(), h); err != nil {
			t.Fatalf("Stop: %v", err)
		}
		// A substrate may retain terminal state (Stopped) or reap the agent
		// immediately (ErrAgentNotFound); both honor the contract.
		st, err := rt.Status(context.Background(), h)
		switch {
		case errors.Is(err, runtime.ErrAgentNotFound):
			// reaped — acceptable.
		case err != nil:
			t.Errorf("Status after Stop = error %v, want Stopped or ErrAgentNotFound", err)
		case st != runtime.StatusStopped:
			t.Errorf("Status after Stop = %q, want stopped (or ErrAgentNotFound)", st)
		}
	})

	if opts.ExecProbe != nil {
		t.Run("Exec runs a command in a started agent", func(t *testing.T) {
			rt := newRuntime()
			h, _ := startAgent(t, rt, "agent-exec")
			res, err := rt.Exec(context.Background(), h, opts.ExecProbe)
			if err != nil {
				t.Fatalf("Exec: unexpected error running %v: %v", opts.ExecProbe, err)
			}
			if res.ExitCode != 0 {
				t.Errorf("Exec probe %v exit = %d (stderr: %q), want 0", opts.ExecProbe, res.ExitCode, res.Stderr)
			}
		})
	}
}
