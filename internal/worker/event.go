// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

// Package worker is the iris fleet worker (ADR-009): a thin Go supervisor that
// subscribes to its NATS subject, drives a pi coding session in an isolated git
// worktree, and streams typed telemetry back on the team activity channel. It
// imports internal/protocol directly — the envelope is a shared dependency, not
// a contract to mirror.
package worker

import (
	"encoding/json"
	"fmt"
)

// Event is one line of pi's `--mode json` output (JSONL). Only the discriminator
// `type` is decoded eagerly; the whole line is kept in Raw so later stages (the
// activity mapping in T3) can pull just the fields they need without this package
// modelling pi's entire event schema — which keeps us resilient to pi adding
// fields within a version.
type Event struct {
	Type string
	Raw  json.RawMessage
}

// pi event "type" values, grounded against pi docs/json.md. A single-shot
// `--mode json` run emits a `session` header, then agent_start → turn_start →
// message_* → tool_execution_* → turn_end, terminating in agent_end.
const (
	EventSession            = "session"
	EventAgentStart         = "agent_start"
	EventAgentEnd           = "agent_end" // terminal event of a single-shot run
	EventTurnStart          = "turn_start"
	EventTurnEnd            = "turn_end"
	EventMessageStart       = "message_start"
	EventMessageUpdate      = "message_update"
	EventMessageEnd         = "message_end"
	EventToolExecutionStart = "tool_execution_start"
	EventToolExecutionEnd   = "tool_execution_end"
)

// decodeEvent parses one JSONL line into an Event. A line that is not valid JSON,
// or that carries no "type", is a protocol violation on pi's side.
func decodeEvent(line []byte) (Event, error) {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(line, &head); err != nil {
		return Event{}, fmt.Errorf("worker: decode pi event: %w", err)
	}
	if head.Type == "" {
		return Event{}, fmt.Errorf("worker: pi event missing %q field: %s", "type", line)
	}
	// Copy the line: bufio.Scanner reuses its buffer between Scan() calls.
	raw := make(json.RawMessage, len(line))
	copy(raw, line)
	return Event{Type: head.Type, Raw: raw}, nil
}
