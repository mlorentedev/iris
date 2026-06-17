// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package worker

import (
	"context"
	"sync"
	"testing"

	"github.com/mlorentedev/iris/internal/protocol"
)

// fakeBus captures published envelopes so tests can assert on the worker's
// telemetry without a real NATS broker. It decodes each payload to prove the
// worker emits well-formed protocol envelopes (not just bytes).
type fakeBus struct {
	mu   sync.Mutex
	sent []busMsg
}

type busMsg struct {
	subject string
	msg     protocol.Message
}

func (b *fakeBus) Publish(subject string, data []byte) error {
	m, err := protocol.Decode(data)
	if err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sent = append(b.sent, busMsg{subject: subject, msg: m})
	return nil
}

func (b *fakeBus) ofType(mt protocol.MessageType) []busMsg {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []busMsg
	for _, m := range b.sent {
		if m.msg.Type == mt {
			out = append(out, m)
		}
	}
	return out
}

// activityEvents returns the decoded ActivityEventPayloads of every activity_event.
func (b *fakeBus) activityEvents(t *testing.T) []protocol.ActivityEventPayload {
	t.Helper()
	var out []protocol.ActivityEventPayload
	for _, m := range b.ofType(protocol.TypeActivityEvent) {
		var p protocol.ActivityEventPayload
		if err := m.msg.Unmarshal(&p); err != nil {
			t.Fatalf("activity_event payload undecodable: %v", err)
		}
		out = append(out, p)
	}
	return out
}

func newTestWorker(t *testing.T, bus Publisher, fixture string) *Worker {
	t.Helper()
	return &Worker{
		Team:  "acme",
		Name:  "worker-1",
		Repo:  initTestRepo(t),
		Base:  t.TempDir(),
		PiBin: buildFakePI(t),
		Env:   fixtureEnv(t, fixture),
		Pub:   bus,
	}
}

func userMsgRaw(t *testing.T, text string) []byte {
	t.Helper()
	msg, err := protocol.NewMessage("user", "worker-1", protocol.TypeUserMessage,
		protocol.UserMessagePayload{Text: text})
	if err != nil {
		t.Fatal(err)
	}
	msg.Context.TeamID = "acme"
	msg.Context.ThreadID = "thread-1"
	raw, err := msg.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// TestHandleHappyPath ties AC1+AC2+AC3 together: a user_message is dispatched,
// pi runs in a worktree, each tool_execution_* event becomes an activity_event,
// and the worktree is cleaned up. The full job returns no error.
func TestHandleHappyPath(t *testing.T) {
	bus := &fakeBus{}
	w := newTestWorker(t, bus, "happy.jsonl")

	if err := w.Handle(context.Background(), userMsgRaw(t, "summarise the readme")); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	acts := bus.activityEvents(t)
	var starts, ends int
	for _, a := range acts {
		switch a.Event {
		case EventToolExecutionStart:
			starts++
		case EventToolExecutionEnd:
			ends++
		}
	}
	if starts != 1 || ends != 1 {
		t.Errorf("tool activity = (%d start, %d end), want (1,1); all=%v", starts, ends, acts)
	}

	// Telemetry must be addressed to the team activity subject and keep threading.
	for _, m := range bus.ofType(protocol.TypeActivityEvent) {
		if m.subject != "team.acme.activity" {
			t.Errorf("activity subject = %q, want team.acme.activity", m.subject)
		}
		if m.msg.Context.ThreadID != "thread-1" {
			t.Errorf("ThreadID = %q, want thread-1 (UI groups by it)", m.msg.Context.ThreadID)
		}
		if m.msg.From != "worker-1" {
			t.Errorf("From = %q, want worker-1", m.msg.From)
		}
	}
}

// TestHandleFailurePath is AC5 (item B): pi exits non-zero, so the worker emits a
// failure activity_event instead of swallowing it, and returns an error.
func TestHandleFailurePath(t *testing.T) {
	bus := &fakeBus{}
	w := newTestWorker(t, bus, "fail.jsonl")
	w.Env = append(w.Env, "FAKEPI_EXIT=1")

	err := w.Handle(context.Background(), userMsgRaw(t, "run a failing command"))
	if err == nil {
		t.Fatal("Handle returned nil for a failed pi run; want an error")
	}

	var failures int
	for _, a := range bus.activityEvents(t) {
		if a.Event == ActivityJobFailed {
			failures++
		}
	}
	if failures != 1 {
		t.Errorf("failure activity_event count = %d, want 1 (failure must be surfaced, not dropped)", failures)
	}
}

// TestEmitStartup is AC3's other half: the worker announces readiness with a
// container_validation envelope when it boots.
func TestEmitStartup(t *testing.T) {
	bus := &fakeBus{}
	w := newTestWorker(t, bus, "happy.jsonl")

	w.EmitStartup(context.Background())

	cv := bus.ofType(protocol.TypeContainerValidation)
	if len(cv) != 1 {
		t.Fatalf("container_validation count = %d, want 1", len(cv))
	}
	var p protocol.ContainerValidationPayload
	if err := cv[0].msg.Unmarshal(&p); err != nil {
		t.Fatalf("container_validation payload undecodable: %v", err)
	}
	if p.Status != "ok" {
		t.Errorf("Status = %q, want ok (pi present + base writable); errors=%v", p.Status, p.Errors)
	}
}
