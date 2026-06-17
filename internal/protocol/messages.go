// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

// Package protocol defines the typed message envelope and subject taxonomy for
// iris's NATS JetStream bus (per ADR-002 and the candidate-nats-protocol-envelope
// pattern). The envelope is the FROZEN wire contract between the Go motor and the
// Go fleet workers (ADR-009); both import this package directly, so the contract
// is a shared dependency, not a cross-language mirror. The JSON fixtures in
// testdata/ are golden tests that pin the wire format against accidental drift.
package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Envelope versioning follows a rolling N-1 window: producers always stamp
// ProtocolVersionCurrent; consumers accept [MinSupported, Current] so a newer
// motor and an older worker interoperate during a rolling deploy. Bumping
// Current requires an ADR amendment plus a deprecation window (see the
// candidate-nats-protocol-envelope pattern).
const (
	ProtocolVersionCurrent      uint8 = 1
	ProtocolVersionMinSupported uint8 = 1
)

// Sentinel errors, comparable with errors.Is.
var (
	// ErrUnsupportedEnvelopeVersion is returned when a message's EnvelopeVersion
	// falls outside [ProtocolVersionMinSupported, ProtocolVersionCurrent].
	ErrUnsupportedEnvelopeVersion = errors.New("protocol: unsupported envelope version")
	// ErrInvalidMessageType is returned for a MessageType outside the taxonomy.
	ErrInvalidMessageType = errors.New("protocol: invalid message type")
)

// MessageType is the envelope discriminator. The taxonomy is deliberately small;
// a new type is added only when an existing one cannot carry the semantics
// without leaking fields to consumers that should not see them.
type MessageType string

// The seven canonical message types.
const (
	TypeUserMessage         MessageType = "user_message"
	TypeLeaderResponse      MessageType = "leader_response"
	TypeSystemCommand       MessageType = "system_command"
	TypeActivityEvent       MessageType = "activity_event"
	TypeContainerValidation MessageType = "container_validation"
	TypeSkillStatus         MessageType = "skill_status"
	TypeMCPStatus           MessageType = "mcp_status"
)

// Validate reports whether t is one of the seven canonical types.
func (t MessageType) Validate() error {
	switch t {
	case TypeUserMessage, TypeLeaderResponse, TypeSystemCommand, TypeActivityEvent,
		TypeContainerValidation, TypeSkillStatus, TypeMCPStatus:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidMessageType, string(t))
	}
}

// Message is the typed envelope every message on the iris NATS bus carries.
// It is flat by design: From, Type and Context.TeamID are read by every
// consumer before it decides whether to handle the message, and Payload stays
// opaque (json.RawMessage) so the envelope is stable as message types grow.
type Message struct {
	EnvelopeVersion uint8           `json:"envelope_version"`
	MessageID       string          `json:"message_id"`
	From            string          `json:"from"`
	To              string          `json:"to"`
	Type            MessageType     `json:"type"`
	Context         MessageContext  `json:"context"`
	RefMessageID    string          `json:"ref_message_id,omitempty"`
	Payload         json.RawMessage `json:"payload"`
	Timestamp       time.Time       `json:"timestamp"`
}

// MessageContext carries threading and routing metadata. TeamID is the tenant
// boundary; ThreadID groups the messages of one logical conversation.
type MessageContext struct {
	ThreadID  string `json:"thread_id"`
	SessionID string `json:"session_id,omitempty"`
	TeamID    string `json:"team_id"`
	TraceID   string `json:"trace_id,omitempty"`
}

// NewMessage builds a Message, stamping the current envelope version, a fresh
// UUID v4, and the producer's UTC timestamp. EnvelopeVersion is never taken
// from caller input. payload is JSON-encoded into the envelope. Set the
// returned message's Context fields (TeamID, ThreadID, ...) before publishing.
func NewMessage(from, to string, mtype MessageType, payload any) (Message, error) {
	if err := mtype.Validate(); err != nil {
		return Message{}, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return Message{}, fmt.Errorf("protocol: marshal %s payload: %w", mtype, err)
	}
	return Message{
		EnvelopeVersion: ProtocolVersionCurrent,
		MessageID:       uuid.NewString(),
		From:            from,
		To:              to,
		Type:            mtype,
		Payload:         raw,
		Timestamp:       time.Now().UTC(),
	}, nil
}

// Marshal encodes the whole envelope to its NATS wire form.
func (m Message) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

// Unmarshal enforces the envelope-version window, then extracts the typed
// payload into `into`. A message outside [MinSupported, Current] is rejected
// with ErrUnsupportedEnvelopeVersion before any payload is touched. A nil
// `into` checks only the version (useful for routing-only consumers).
func (m Message) Unmarshal(into any) error {
	if m.EnvelopeVersion < ProtocolVersionMinSupported || m.EnvelopeVersion > ProtocolVersionCurrent {
		return fmt.Errorf("%w: got %d, supported [%d,%d]", ErrUnsupportedEnvelopeVersion,
			m.EnvelopeVersion, ProtocolVersionMinSupported, ProtocolVersionCurrent)
	}
	if into == nil {
		return nil
	}
	return json.Unmarshal(m.Payload, into)
}

// Decode parses NATS wire bytes into a Message. It does not validate the
// version (that happens on Unmarshal, when the consumer reads the payload) so a
// router can inspect Type and Context without rejecting a near-version message.
func Decode(data []byte) (Message, error) {
	var m Message
	if err := json.Unmarshal(data, &m); err != nil {
		return Message{}, fmt.Errorf("protocol: decode envelope: %w", err)
	}
	return m, nil
}
