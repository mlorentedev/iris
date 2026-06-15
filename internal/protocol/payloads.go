// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package protocol

import "encoding/json"

// Payload structs for the seven message types. Each is the body carried in
// Message.Payload (json.RawMessage); consumers select the struct by Message.Type.
// Fields grow additively (new optional fields) within an envelope version; a
// breaking change is a new MessageType, never a fork of an existing payload.

// UserMessagePayload carries user input from the frontend to a team's leader.
type UserMessagePayload struct {
	Text        string   `json:"text"`
	Attachments []string `json:"attachments,omitempty"`
	Command     string   `json:"command,omitempty"`
}

// LeaderResponsePayload is the leader agent's reply back to the UI.
type LeaderResponsePayload struct {
	Text         string   `json:"text"`
	Blocks       []string `json:"blocks,omitempty"`
	FinishReason string   `json:"finish_reason"`
}

// SystemCommandPayload is a lifecycle/control command from the motor to an agent.
type SystemCommandPayload struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

// ActivityEventPayload is a streaming activity event for the live UI. Data
// stays opaque so new event shapes don't force a payload change.
type ActivityEventPayload struct {
	Event   string          `json:"event"`
	Target  string          `json:"target,omitempty"`
	Summary string          `json:"summary,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// ContainerValidationPayload is the readiness signal from a freshly-deployed
// agent container.
type ContainerValidationPayload struct {
	Status string   `json:"status"`
	Checks []string `json:"checks"`
	Errors []string `json:"errors,omitempty"`
}

// SkillStatusPayload is a skill install/load/failure event.
type SkillStatusPayload struct {
	Skill  string `json:"skill"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// MCPStatusPayload is an MCP server connection/disconnection/error event.
type MCPStatusPayload struct {
	Server string `json:"server"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}
