// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package worker

import (
	"testing"

	"github.com/mlorentedev/iris/internal/protocol"
)

// TestDispatchDecodesUserMessage is AC1: a user_message envelope built with the
// shared internal/protocol package decodes back into a Job — no mirror, no
// hand-rolled JSON. This is the roundtrip the proposal calls out.
func TestDispatchDecodesUserMessage(t *testing.T) {
	msg, err := protocol.NewMessage("user", "worker-1", protocol.TypeUserMessage,
		protocol.UserMessagePayload{Text: "add a license header to main.go"})
	if err != nil {
		t.Fatalf("NewMessage: %v", err)
	}
	msg.Context.TeamID = "acme"
	raw, err := msg.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	job, err := DecodeJob(raw)
	if err != nil {
		t.Fatalf("DecodeJob: %v", err)
	}
	if job.Prompt != "add a license header to main.go" {
		t.Errorf("Prompt = %q, want the envelope text", job.Prompt)
	}
	if job.TeamID != "acme" {
		t.Errorf("TeamID = %q, want %q", job.TeamID, "acme")
	}
	if job.MessageID == "" {
		t.Error("MessageID is empty; want the envelope's id (worktrees and dedup key off it)")
	}
}

// TestDispatchRejectsWrongType: the worker subscribes to exactly one job type.
// Anything else on its subject is a routing bug and must error, not run pi.
func TestDispatchRejectsWrongType(t *testing.T) {
	msg, err := protocol.NewMessage("motor", "worker-1", protocol.TypeSystemCommand,
		protocol.SystemCommandPayload{Command: "noop"})
	if err != nil {
		t.Fatalf("NewMessage: %v", err)
	}
	raw, err := msg.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if _, err := DecodeJob(raw); err == nil {
		t.Error("DecodeJob accepted a system_command; want an error (wrong type for this subject)")
	}
}

// TestDispatchRejectsEmptyPrompt: an empty prompt is nothing for pi to do —
// reject it at the door rather than spawning a no-op coding session.
func TestDispatchRejectsEmptyPrompt(t *testing.T) {
	msg, err := protocol.NewMessage("user", "worker-1", protocol.TypeUserMessage,
		protocol.UserMessagePayload{Text: "   "})
	if err != nil {
		t.Fatalf("NewMessage: %v", err)
	}
	raw, err := msg.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if _, err := DecodeJob(raw); err == nil {
		t.Error("DecodeJob accepted a blank prompt; want an error")
	}
}
