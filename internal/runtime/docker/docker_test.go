// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package docker

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"

	"github.com/mlorentedev/iris/internal/runtime"
	"github.com/mlorentedev/iris/internal/runtime/runtimetest"
)

func TestMapState(t *testing.T) {
	cases := []struct {
		status container.ContainerState
		exit   int
		want   runtime.AgentStatus
	}{
		{container.StateCreated, 0, runtime.StatusStarting},
		{container.StateRestarting, 0, runtime.StatusStarting},
		{container.StateRunning, 0, runtime.StatusRunning},
		{container.StatePaused, 0, runtime.StatusRunning},
		{container.StateRemoving, 0, runtime.StatusStopping},
		{container.StateExited, 0, runtime.StatusStopped},
		{container.StateExited, 137, runtime.StatusFailed},
		{container.StateDead, 0, runtime.StatusFailed},
		{"some-future-state", 0, runtime.StatusFailed},
	}
	for _, c := range cases {
		got := mapState(&container.State{Status: c.status, ExitCode: c.exit})
		if got != c.want {
			t.Errorf("mapState(%q, exit=%d) = %q, want %q", c.status, c.exit, got, c.want)
		}
	}
}

func TestContainerName(t *testing.T) {
	if got := containerName("alpha", "team-1"); got != "iris-team-1-alpha" {
		t.Errorf("containerName = %q, want iris-team-1-alpha", got)
	}
}

func TestEnvSlice(t *testing.T) {
	got := envSlice(map[string]string{"B": "2", "A": "1"})
	if len(got) != 2 || got[0] != "A=1" || got[1] != "B=2" {
		t.Errorf("envSlice = %v, want [A=1 B=2] (sorted)", got)
	}
	if envSlice(nil) != nil {
		t.Error("envSlice(nil) must be nil")
	}
}

// testImage is a tiny image with both `sleep` (to keep the agent alive) and
// `true` (the exec probe). Pinned for reproducibility.
const testImage = "busybox:1.37"

// requireDaemon returns a live Docker adapter or skips the test.
func requireDaemon(t *testing.T) *Runtime {
	t.Helper()
	rt, err := New()
	if err != nil {
		t.Skipf("docker client unavailable: %v", err)
	}
	if _, err := rt.cli.Ping(context.Background()); err != nil {
		_ = rt.Close()
		t.Skipf("docker daemon unavailable: %v", err)
	}
	return rt
}

// cleanupConfTeam removes any residue from a crashed prior run (the conformance
// suite normally tears down its own agents via t.Cleanup) and schedules the same
// sweep after the test. "team-conf" mirrors runtimetest's conformance team.
func cleanupConfTeam(t *testing.T, rt *Runtime) {
	t.Helper()
	sweep := func() {
		f := filters.NewArgs(
			filters.Arg("label", labelManaged+"=true"),
			filters.Arg("label", labelTeam+"=team-conf"),
		)
		list, err := rt.cli.ContainerList(context.Background(), container.ListOptions{All: true, Filters: f})
		if err != nil {
			return
		}
		for i := range list {
			_ = rt.cli.ContainerRemove(context.Background(), list[i].ID, container.RemoveOptions{Force: true})
		}
	}
	sweep()
	t.Cleanup(sweep)
}

// TestDockerConformance runs the shared AgentRuntime conformance suite against a
// real Docker daemon. It skips (does not fail) when no daemon or registry is
// reachable, so the package still builds and unit-tests in restricted CI.
func TestDockerConformance(t *testing.T) {
	rt := requireDaemon(t)
	t.Cleanup(func() { _ = rt.Close() })

	if err := rt.pull(context.Background(), testImage); err != nil {
		t.Skipf("cannot pull %s (no network?): %v", testImage, err)
	}
	cleanupConfTeam(t, rt)

	runtimetest.RunConformance(t, func() runtime.AgentRuntime { return rt }, runtimetest.Options{
		NewSpec: func(name, teamID string) runtime.AgentSpec {
			return runtime.AgentSpec{
				Name:      name,
				TeamID:    teamID,
				Image:     testImage,
				Command:   []string{"sleep", "3600"},
				Harness:   runtime.HarnessPi,
				Resources: runtime.ResourceRequest{CPUMillis: 250, MemoryBytes: 64 << 20},
			}
		},
		ExecProbe: []string{"true"},
	})
}
