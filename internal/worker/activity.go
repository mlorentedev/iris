// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package worker

import (
	"encoding/json"
	"fmt"

	"github.com/mlorentedev/iris/internal/protocol"
)

// toolEvent is the subset of a pi tool_execution_* event the activity mapper
// reads. The rest of the line stays opaque and rides along in the activity
// payload's Data field.
type toolEvent struct {
	ToolCallID string          `json:"toolCallId"`
	ToolName   string          `json:"toolName"`
	Args       json.RawMessage `json:"args"`
	Result     json.RawMessage `json:"result"`
	IsError    bool            `json:"isError"`
}

// ActivityFor maps a pi event to an iris activity_event payload, returning
// ok=false for events the skeleton does not surface. AC3 surfaces exactly one
// activity_event per pi tool_execution_* event.
//
// This is the FROZEN-surface check the proposal flagged: pi's tool events map
// cleanly onto protocol.ActivityEventPayload — Event/Target/Summary are the
// human-facing triple and the tool's own args/result ride along untouched in the
// opaque Data field. No new field on the FROZEN payload was required, so no ADR
// amendment is needed for the skeleton.
func ActivityFor(e Event) (protocol.ActivityEventPayload, bool) {
	switch e.Type {
	case EventToolExecutionStart, EventToolExecutionEnd:
		var t toolEvent
		// e.Raw is already validated JSON (decodeEvent parsed its type), so a
		// well-formed event never errors here; a partial event just yields zero
		// fields, which is acceptable for telemetry.
		_ = json.Unmarshal(e.Raw, &t)

		payload := protocol.ActivityEventPayload{
			Event:  e.Type,
			Target: t.ToolName,
		}
		if e.Type == EventToolExecutionStart {
			payload.Summary = fmt.Sprintf("running %s", t.ToolName)
			payload.Data = t.Args
		} else {
			if t.IsError {
				payload.Summary = fmt.Sprintf("%s failed", t.ToolName)
			} else {
				payload.Summary = fmt.Sprintf("completed %s", t.ToolName)
			}
			payload.Data = t.Result
		}
		return payload, true
	default:
		return protocol.ActivityEventPayload{}, false
	}
}
