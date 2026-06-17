// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package worker

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

// buildFakePI compiles the testdata fake-pi stand-in into a temp dir and returns
// its path. Compiling on demand (rather than committing a binary) keeps the
// harness cross-platform and never ships the stand-in.
func buildFakePI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "fakepi")
	cmd := exec.Command("go", "build", "-o", bin, "./testdata/fakepi")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build fakepi: %v", err)
	}
	return bin
}

// fixtureEnv returns an environment that points fake-pi at the named fixture.
func fixtureEnv(t *testing.T, name string) []string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return append(os.Environ(), "FAKEPI_FIXTURE="+abs)
}

// TestDriverStreamsEventsToAgentEnd is the keystone of T1: it pins pi's
// `--mode json` contract via fake-pi. The driver must spawn the subprocess,
// deliver every event line to the handler, recognise the terminal agent_end,
// and report a clean exit.
func TestDriverStreamsEventsToAgentEnd(t *testing.T) {
	d := &Driver{
		Bin: buildFakePI(t),
		Dir: t.TempDir(),
		Env: fixtureEnv(t, "happy.jsonl"),
	}

	var got []string
	res, err := d.Run(context.Background(), "summarise the readme", func(e Event) {
		got = append(got, e.Type)
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
	if !res.SawAgentEnd {
		t.Errorf("SawAgentEnd = false, want true (terminal event must be observed)")
	}

	// The tool-execution pair and the terminal event must all reach the handler.
	for _, want := range []string{EventToolExecutionStart, EventToolExecutionEnd, EventAgentEnd} {
		if !slices.Contains(got, want) {
			t.Errorf("event %q was not delivered to the handler; got sequence %v", want, got)
		}
	}

	// agent_end is terminal: it must be the last event the handler sees.
	if n := len(got); n == 0 || got[n-1] != EventAgentEnd {
		t.Errorf("last delivered event = %q, want %q (full sequence %v)", last(got), EventAgentEnd, got)
	}
}

func last(xs []string) string {
	if len(xs) == 0 {
		return ""
	}
	return xs[len(xs)-1]
}
