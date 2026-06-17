// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package worker

import (
	"fmt"
	"strings"

	"github.com/mlorentedev/iris/internal/protocol"
)

// Job is one unit of work decoded from a user_message envelope. It carries the
// fields the worker needs downstream: the prompt for pi, the team (to address
// activity events back), and the MessageID (the per-job worktree name and the
// future dedup key). The full Envelope is retained for tracing/threading.
type Job struct {
	MessageID string
	TeamID    string
	Prompt    string
	Envelope  protocol.Message
}

// DecodeJob turns a raw NATS message body into a Job using the shared protocol
// package — the envelope is a dependency, not a contract to re-implement (ADR-009).
// It rejects anything that is not a user_message with a non-empty prompt: the
// skeleton worker has exactly one job type, and a different type on its subject
// is a routing bug that must surface rather than silently spawn pi.
func DecodeJob(raw []byte) (Job, error) {
	msg, err := protocol.Decode(raw)
	if err != nil {
		return Job{}, err
	}
	if msg.Type != protocol.TypeUserMessage {
		return Job{}, fmt.Errorf("worker: unexpected message type %q, want %q",
			msg.Type, protocol.TypeUserMessage)
	}

	var p protocol.UserMessagePayload
	if err := msg.Unmarshal(&p); err != nil {
		return Job{}, fmt.Errorf("worker: decode user_message payload: %w", err)
	}
	if strings.TrimSpace(p.Text) == "" {
		return Job{}, fmt.Errorf("worker: user_message %s has empty prompt text", msg.MessageID)
	}

	return Job{
		MessageID: msg.MessageID,
		TeamID:    msg.Context.TeamID,
		Prompt:    p.Text,
		Envelope:  msg,
	}, nil
}
