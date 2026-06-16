// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package runtime_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mlorentedev/iris/internal/runtime"
	"github.com/mlorentedev/iris/internal/runtime/runtimetest"
)

// fakeRuntime is an in-memory AgentRuntime used to exercise the conformance
// suite and prove the FROZEN contract is implementable. It is a test fixture,
// not a shippable substrate.
type fakeRuntime struct {
	agents map[string]*fakeAgent // keyed by identity(name, teamID)
}

type fakeAgent struct {
	handle   runtime.AgentHandle
	specHash string
	status   runtime.AgentStatus
}

func newFakeConcrete() *fakeRuntime { return &fakeRuntime{agents: map[string]*fakeAgent{}} }
func newFake() runtime.AgentRuntime { return newFakeConcrete() }

func identity(name, teamID string) string { return name + "\x00" + teamID }

func (f *fakeRuntime) Start(_ context.Context, spec runtime.AgentSpec) (runtime.AgentHandle, error) {
	key := identity(spec.Name, spec.TeamID)
	if a, ok := f.agents[key]; ok {
		if a.specHash != spec.SpecHash() {
			return runtime.AgentHandle{}, runtime.ErrSpecDrift
		}
		return a.handle, nil // idempotent: same identity, same spec
	}
	h := runtime.AgentHandle{
		ID:        "fake-" + key,
		Name:      spec.Name,
		TeamID:    spec.TeamID,
		Substrate: runtime.SubstrateLocal,
	}
	f.agents[key] = &fakeAgent{handle: h, specHash: spec.SpecHash(), status: runtime.StatusRunning}
	return h, nil
}

func (f *fakeRuntime) Stop(_ context.Context, h runtime.AgentHandle) error {
	if a, ok := f.agents[identity(h.Name, h.TeamID)]; ok && a.handle.ID == h.ID {
		a.status = runtime.StatusStopped
	}
	return nil // idempotent; unknown handle is a no-op
}

func (f *fakeRuntime) Status(_ context.Context, h runtime.AgentHandle) (runtime.AgentStatus, error) {
	if a, ok := f.agents[identity(h.Name, h.TeamID)]; ok && a.handle.ID == h.ID {
		return a.status, nil
	}
	return "", runtime.ErrAgentNotFound
}

func (f *fakeRuntime) Exec(_ context.Context, h runtime.AgentHandle, cmd []string) (runtime.ExecResult, error) {
	if _, ok := f.agents[identity(h.Name, h.TeamID)]; !ok {
		return runtime.ExecResult{}, runtime.ErrAgentNotFound
	}
	if len(cmd) > 0 && cmd[0] == "false" {
		return runtime.ExecResult{ExitCode: 1}, nil
	}
	return runtime.ExecResult{ExitCode: 0, Stdout: strings.Join(cmd, " ")}, nil
}

func fakeSpec(name, teamID string) runtime.AgentSpec {
	return runtime.AgentSpec{
		Name:      name,
		TeamID:    teamID,
		Image:     "iris/agent-pi:test",
		Harness:   runtime.HarnessPi,
		Env:       map[string]string{"PI_PROVIDER": "ollama"},
		Resources: runtime.ResourceRequest{CPUMillis: 500, MemoryBytes: 512 << 20},
	}
}

// TestFakeRuntimeConformance proves the base contract is satisfiable and, in
// doing so, exercises the conformance suite that every real substrate will run.
func TestFakeRuntimeConformance(t *testing.T) {
	runtimetest.RunConformance(t, newFake, runtimetest.Options{
		NewSpec:   fakeSpec,
		ExecProbe: []string{"true"},
	})
}

func TestHarnessValidate(t *testing.T) {
	valid := []runtime.Harness{runtime.HarnessPi, runtime.HarnessOpenCode, runtime.HarnessHermes}
	for _, h := range valid {
		if err := h.Validate(); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", h, err)
		}
	}
	// "claude" is rejected by policy (ToS) like any other unknown value.
	for _, h := range []runtime.Harness{"", "claude", "ollama", "bogus"} {
		if err := h.Validate(); !errors.Is(err, runtime.ErrInvalidHarness) {
			t.Errorf("Validate(%q) = %v, want ErrInvalidHarness", h, err)
		}
	}
}

func TestAgentStatusValidate(t *testing.T) {
	valid := []runtime.AgentStatus{
		runtime.StatusStarting, runtime.StatusRunning,
		runtime.StatusStopping, runtime.StatusStopped, runtime.StatusFailed,
	}
	for _, s := range valid {
		if err := s.Validate(); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", s, err)
		}
		if s.String() != string(s) {
			t.Errorf("String(%q) = %q, want %q", s, s.String(), string(s))
		}
	}
	for _, s := range []runtime.AgentStatus{"", "paused", "RUNNING"} {
		if err := s.Validate(); !errors.Is(err, runtime.ErrInvalidStatus) {
			t.Errorf("Validate(%q) = %v, want ErrInvalidStatus", s, err)
		}
	}
}

func TestSpecHashDeterministicAndOrderIndependent(t *testing.T) {
	a := fakeSpec("alpha", "team-1")
	a.Env = map[string]string{"A": "1", "B": "2", "C": "3"}
	b := fakeSpec("alpha", "team-1")
	b.Env = map[string]string{"C": "3", "A": "1", "B": "2"} // different insertion order
	if a.SpecHash() != b.SpecHash() {
		t.Errorf("SpecHash must be order-independent over Env: %s != %s", a.SpecHash(), b.SpecHash())
	}
	if first, second := a.SpecHash(), a.SpecHash(); first != second {
		t.Errorf("SpecHash must be deterministic across calls: %s != %s", first, second)
	}
}

func TestSpecHashMutableFieldsDrift(t *testing.T) {
	base := fakeSpec("alpha", "team-1")
	mutators := map[string]func(*runtime.AgentSpec){
		"Image":       func(s *runtime.AgentSpec) { s.Image = "other:tag" },
		"Command":     func(s *runtime.AgentSpec) { s.Command = []string{"sh", "-c", "echo hi"} },
		"Harness":     func(s *runtime.AgentSpec) { s.Harness = runtime.HarnessOpenCode },
		"Env value":   func(s *runtime.AgentSpec) { s.Env = map[string]string{"PI_PROVIDER": "openrouter"} },
		"Env key":     func(s *runtime.AgentSpec) { s.Env = map[string]string{"OTHER": "ollama"} },
		"CPUMillis":   func(s *runtime.AgentSpec) { s.Resources.CPUMillis = 1000 },
		"MemoryBytes": func(s *runtime.AgentSpec) { s.Resources.MemoryBytes = 1 << 30 },
		"GPUs":        func(s *runtime.AgentSpec) { s.Resources.GPUs = 1 },
	}
	for name, mutate := range mutators {
		drift := base
		mutate(&drift)
		if drift.SpecHash() == base.SpecHash() {
			t.Errorf("mutating %s must change SpecHash (it is part of drift)", name)
		}
	}
}

func TestSpecHashExcludesIdentity(t *testing.T) {
	base := fakeSpec("alpha", "team-1")
	renamed := base
	renamed.Name = "beta"
	renamed.TeamID = "team-2"
	if renamed.SpecHash() != base.SpecHash() {
		t.Error("SpecHash must exclude identity (Name, TeamID): changing them is a different agent, not drift")
	}
}

// fakeAllCaps opts into every capability, so the As* accessors can be tested in
// both directions: a base runtime (no capabilities) and a fully capable one.
type fakeAllCaps struct{ *fakeRuntime }

func (fakeAllCaps) EnsureOllama(context.Context, string) error         { return nil }
func (fakeAllCaps) ListOllamaModels(context.Context) ([]string, error) { return []string{"qwen"}, nil }
func (fakeAllCaps) EnsureQdrant(context.Context) (runtime.QdrantEndpoint, error) {
	return runtime.QdrantEndpoint{}, nil
}
func (fakeAllCaps) QdrantCollections(context.Context) ([]string, error) { return nil, nil }
func (fakeAllCaps) EnsureTeamNetwork(context.Context, string) error     { return nil }
func (fakeAllCaps) AttachToTeamNetwork(context.Context, runtime.AgentHandle, string) error {
	return nil
}
func (fakeAllCaps) DeploySidecar(context.Context, runtime.AgentHandle, runtime.SidecarSpec) error {
	return nil
}

func TestCapabilityAccessors(t *testing.T) {
	base := newFake()
	capable := fakeAllCaps{newFakeConcrete()}

	cases := []struct {
		name string
		as   func(runtime.AgentRuntime) error // returns the accessor's error
	}{
		{"OllamaManager", func(rt runtime.AgentRuntime) error { _, e := runtime.AsOllamaManager(rt); return e }},
		{"QdrantManager", func(rt runtime.AgentRuntime) error { _, e := runtime.AsQdrantManager(rt); return e }},
		{"NetworkManager", func(rt runtime.AgentRuntime) error { _, e := runtime.AsNetworkManager(rt); return e }},
		{"SidecarManager", func(rt runtime.AgentRuntime) error { _, e := runtime.AsSidecarManager(rt); return e }},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/present", func(t *testing.T) {
			if err := tc.as(capable); err != nil {
				t.Errorf("accessor on a capable runtime returned error: %v", err)
			}
		})
		t.Run(tc.name+"/missing", func(t *testing.T) {
			err := tc.as(base)
			if err == nil {
				t.Fatal("accessor on a base runtime must fail")
			}
			// agent-oriented error names its own missing capability (ERROR/WHY/FIX).
			for _, want := range []string{"ERROR:", "WHY:", "FIX:", tc.name} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("missing-capability error missing %q:\n%v", want, err)
				}
			}
		})
	}
}
