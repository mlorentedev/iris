// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package protocol

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

// allTypes is the canonical taxonomy; tests assert it stays at seven.
var allTypes = []MessageType{
	TypeUserMessage, TypeLeaderResponse, TypeSystemCommand, TypeActivityEvent,
	TypeContainerValidation, TypeSkillStatus, TypeMCPStatus,
}

func TestMessageType_Validate(t *testing.T) {
	if len(allTypes) != 7 {
		t.Fatalf("taxonomy must stay at 7 types, got %d", len(allTypes))
	}
	for _, mt := range allTypes {
		if err := mt.Validate(); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", mt, err)
		}
	}
	for _, bad := range []MessageType{"", "user", "UserMessage", "activity"} {
		if err := bad.Validate(); !errors.Is(err, ErrInvalidMessageType) {
			t.Errorf("Validate(%q) = %v, want ErrInvalidMessageType", bad, err)
		}
	}
}

func TestNewMessage_StampsEnvelope(t *testing.T) {
	m, err := NewMessage("user", "leader", TypeUserMessage, UserMessagePayload{Text: "hi"})
	if err != nil {
		t.Fatalf("NewMessage: %v", err)
	}
	if m.EnvelopeVersion != ProtocolVersionCurrent {
		t.Errorf("EnvelopeVersion = %d, want %d (never from caller)", m.EnvelopeVersion, ProtocolVersionCurrent)
	}
	if _, err := uuid.Parse(m.MessageID); err != nil {
		t.Errorf("MessageID %q is not a valid UUID: %v", m.MessageID, err)
	}
	if m.Timestamp.IsZero() {
		t.Error("Timestamp not stamped")
	}
	if _, err := NewMessage("a", "b", MessageType("bogus"), nil); !errors.Is(err, ErrInvalidMessageType) {
		t.Error("NewMessage with an invalid type must fail with ErrInvalidMessageType")
	}
}

// TestRoundTripEveryType exercises the full chain for each type:
// NewMessage -> Marshal -> Decode -> Unmarshal, and asserts the payload survives.
func TestRoundTripEveryType(t *testing.T) {
	cases := []struct {
		mtype   MessageType
		payload any
		into    any // pointer to a fresh payload struct
	}{
		{TypeUserMessage, UserMessagePayload{Text: "build a login page", Command: "chat"}, &UserMessagePayload{}},
		{TypeLeaderResponse, LeaderResponsePayload{Text: "on it", FinishReason: "stop"}, &LeaderResponsePayload{}},
		{TypeSystemCommand, SystemCommandPayload{Command: "reload_skills", Args: []string{"frontend"}}, &SystemCommandPayload{}},
		{TypeActivityEvent, ActivityEventPayload{Event: "tool_call_started", Target: "write_file"}, &ActivityEventPayload{}},
		{TypeContainerValidation, ContainerValidationPayload{Status: "ready", Checks: []string{"nats"}}, &ContainerValidationPayload{}},
		{TypeSkillStatus, SkillStatusPayload{Skill: "frontend", Status: "loaded"}, &SkillStatusPayload{}},
		{TypeMCPStatus, MCPStatusPayload{Server: "drawio", Status: "connected"}, &MCPStatusPayload{}},
	}
	if len(cases) != len(allTypes) {
		t.Fatalf("round-trip table covers %d types, taxonomy has %d", len(cases), len(allTypes))
	}

	for _, tc := range cases {
		t.Run(string(tc.mtype), func(t *testing.T) {
			m, err := NewMessage("from", "to", tc.mtype, tc.payload)
			if err != nil {
				t.Fatalf("NewMessage: %v", err)
			}
			wire, err := m.Marshal()
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			got, err := Decode(wire)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if got.Type != tc.mtype {
				t.Errorf("decoded Type = %q, want %q", got.Type, tc.mtype)
			}
			if err := got.Unmarshal(tc.into); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if back := reflect.ValueOf(tc.into).Elem().Interface(); !reflect.DeepEqual(back, tc.payload) {
				t.Errorf("payload round-trip mismatch:\n got  %#v\n want %#v", back, tc.payload)
			}
		})
	}
}

// TestUnmarshal_VersionWindow verifies the rolling-N-1 acceptance window.
func TestUnmarshal_VersionWindow(t *testing.T) {
	base, err := NewMessage("user", "leader", TypeUserMessage, UserMessagePayload{Text: "hi"})
	if err != nil {
		t.Fatalf("NewMessage: %v", err)
	}

	tests := []struct {
		name    string
		version uint8
		wantErr bool
	}{
		{"current accepted", ProtocolVersionCurrent, false},
		{"min supported accepted", ProtocolVersionMinSupported, false},
		{"zero rejected", 0, true},
		{"above current rejected", ProtocolVersionCurrent + 1, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := base
			m.EnvelopeVersion = tc.version
			err := m.Unmarshal(&UserMessagePayload{})
			if tc.wantErr {
				if !errors.Is(err, ErrUnsupportedEnvelopeVersion) {
					t.Errorf("version %d: err = %v, want ErrUnsupportedEnvelopeVersion", tc.version, err)
				}
			} else if err != nil {
				t.Errorf("version %d: err = %v, want nil", tc.version, err)
			}
		})
	}
}

func TestSubjectFor(t *testing.T) {
	ok := []struct {
		team, worker string
		kind         SubjectKind
		want         string
	}{
		{"acme", "", SubjectLeader, "team.acme.leader"},
		{"acme", "", SubjectActivity, "team.acme.activity"},
		{"acme", "", SubjectSystem, "team.acme.system"},
		{"acme", "code-writer", SubjectWorker, "team.acme.worker.code-writer"},
	}
	for _, tc := range ok {
		var (
			got string
			err error
		)
		if tc.kind == SubjectWorker {
			got, err = SubjectFor(tc.team, tc.kind, tc.worker)
		} else {
			got, err = SubjectFor(tc.team, tc.kind)
		}
		if err != nil || got != tc.want {
			t.Errorf("SubjectFor(%q,%q,%q) = (%q,%v), want (%q,nil)", tc.team, tc.kind, tc.worker, got, err, tc.want)
		}
	}

	// Reserved NATS chars, casing, hyphen edges, underscores, too-short names.
	for _, bad := range []string{"Acme", "ac me", "ac.me", "ac*", "ac>", "ac_me", "-acme", "acme-", "", "a"} {
		if _, err := SubjectFor(bad, SubjectLeader); !errors.Is(err, ErrInvalidName) {
			t.Errorf("SubjectFor(%q, leader) = %v, want ErrInvalidName", bad, err)
		}
	}
	// A bad worker name is rejected too.
	if _, err := SubjectFor("acme", SubjectWorker, "bad.name"); !errors.Is(err, ErrInvalidName) {
		t.Error("worker name with a dot must be rejected")
	}
	// Worker kind needs exactly one name; the others take none.
	if _, err := SubjectFor("acme", SubjectWorker); !errors.Is(err, ErrInvalidName) {
		t.Error("worker kind without a name must error")
	}
	if _, err := SubjectFor("acme", SubjectLeader, "extra"); !errors.Is(err, ErrInvalidSubjectKind) {
		t.Error("non-worker kind with a worker name must error")
	}
	if _, err := SubjectFor("acme", SubjectKind("bogus")); !errors.Is(err, ErrInvalidSubjectKind) {
		t.Error("unknown subject kind must error")
	}
}

// TestWireFixtures loads the committed JSON oracle shared with the Python
// workers and asserts every fixture decodes, validates, and yields its payload.
func TestWireFixtures(t *testing.T) {
	raw, err := os.ReadFile("testdata/wire-fixtures.json")
	if err != nil {
		t.Fatalf("read fixtures: %v", err)
	}
	var fixtures []json.RawMessage
	if err := json.Unmarshal(raw, &fixtures); err != nil {
		t.Fatalf("parse fixtures array: %v", err)
	}
	if len(fixtures) != len(allTypes) {
		t.Fatalf("fixtures cover %d messages, taxonomy has %d types", len(fixtures), len(allTypes))
	}

	seen := map[MessageType]bool{}
	for i, fx := range fixtures {
		m, err := Decode(fx)
		if err != nil {
			t.Fatalf("fixture %d: decode: %v", i, err)
		}
		if err := m.Type.Validate(); err != nil {
			t.Errorf("fixture %d: type %q invalid: %v", i, m.Type, err)
		}
		if err := m.Unmarshal(nil); err != nil {
			t.Errorf("fixture %d: version check failed: %v", i, err)
		}
		if m.Context.TeamID == "" {
			t.Errorf("fixture %d (%s): empty team_id (tenant boundary)", i, m.Type)
		}
		seen[m.Type] = true
	}
	for _, mt := range allTypes {
		if !seen[mt] {
			t.Errorf("no wire fixture for message type %q", mt)
		}
	}
}
