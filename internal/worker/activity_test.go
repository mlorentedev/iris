// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package worker

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestActivityForToolStart: a pi tool_execution_start maps to one activity_event
// whose Target is the tool and whose Data carries the call args verbatim (opaque
// pass-through — the UI decides how to render them).
func TestActivityForToolStart(t *testing.T) {
	ev, err := decodeEvent([]byte(`{"type":"tool_execution_start","toolCallId":"c1","toolName":"read_file","args":{"path":"README.md"}}`))
	if err != nil {
		t.Fatal(err)
	}
	a, ok := ActivityFor(ev)
	if !ok {
		t.Fatal("tool_execution_start did not map to an activity_event; AC3 wants one per tool_execution_* event")
	}
	if a.Event != EventToolExecutionStart {
		t.Errorf("Event = %q, want %q", a.Event, EventToolExecutionStart)
	}
	if a.Target != "read_file" {
		t.Errorf("Target = %q, want %q", a.Target, "read_file")
	}
	if a.Summary == "" {
		t.Error("Summary is empty; want a human-readable line")
	}
	// Data is the opaque args blob, parseable back to the original object.
	var got map[string]any
	if err := json.Unmarshal(a.Data, &got); err != nil {
		t.Fatalf("Data is not valid JSON: %v", err)
	}
	if got["path"] != "README.md" {
		t.Errorf("Data.path = %v, want README.md", got["path"])
	}
}

// TestActivityForToolEndError: a failed tool execution maps to an activity_event
// whose Summary makes the failure legible (item B feeds off this distinction).
func TestActivityForToolEndError(t *testing.T) {
	ev, err := decodeEvent([]byte(`{"type":"tool_execution_end","toolName":"bash","result":{"exitCode":1},"isError":true}`))
	if err != nil {
		t.Fatal(err)
	}
	a, ok := ActivityFor(ev)
	if !ok {
		t.Fatal("tool_execution_end did not map to an activity_event")
	}
	if a.Target != "bash" {
		t.Errorf("Target = %q, want %q", a.Target, "bash")
	}
	if !strings.Contains(strings.ToLower(a.Summary), "fail") {
		t.Errorf("Summary = %q, want it to signal failure", a.Summary)
	}
}

// TestActivityForIgnoredEvent: the skeleton surfaces tool_execution_* only (AC3).
// Lifecycle chatter like turn_start must NOT become an activity_event.
func TestActivityForIgnoredEvent(t *testing.T) {
	ev, err := decodeEvent([]byte(`{"type":"turn_start"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ActivityFor(ev); ok {
		t.Error("turn_start mapped to an activity_event; want it ignored")
	}
}
