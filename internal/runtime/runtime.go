// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Sentinel errors, comparable with errors.Is.
var (
	// ErrSpecDrift is returned by Start when an agent with the same identity
	// (Name + TeamID) is already running but its spec differs from the one
	// passed in. The caller must reconcile explicitly — Stop + Start for a full
	// restart — rather than have Start silently re-create. Silent re-create was
	// rejected as a production footgun: operators have run stale images for days
	// because Start "worked" without surfacing the drift.
	ErrSpecDrift = errors.New("runtime: agent spec drift; explicit Stop+Start required")

	// ErrAgentNotFound is returned by Status, Exec or Stop when the handle does
	// not refer to an agent this runtime knows about.
	ErrAgentNotFound = errors.New("runtime: agent not found")

	// ErrInvalidHarness is returned for a Harness outside the taxonomy.
	ErrInvalidHarness = errors.New("runtime: invalid harness")

	// ErrInvalidStatus is returned when an AgentStatus is outside the lifecycle
	// enum (e.g. decoded from an untrusted source).
	ErrInvalidStatus = errors.New("runtime: invalid agent status")
)

// AgentRuntime is the FROZEN base every substrate adapter (Docker, Kubernetes,
// local-process) MUST satisfy. It captures the universal agent lifecycle and
// nothing substrate-specific: volume, network, sidecar and vector-store
// management live in optional capability sub-interfaces (see capability_*.go)
// that a substrate opts into. The interface is deliberately substrate-neutral —
// per ADR-005, no method here may exist only to suit one substrate's shape.
type AgentRuntime interface {
	// Start provisions resources and launches the agent.
	//
	// Idempotency: Start is idempotent for the (Name, TeamID) identity — calling
	// Start again for an agent that is already running returns the existing
	// AgentHandle without re-launching, IF AND ONLY IF the spec's SpecHash
	// matches the running agent's. If the spec differs (different Image, Command,
	// Harness, Env, Resources), Start returns ErrSpecDrift and the caller must
	// reconcile with Stop + Start.
	Start(ctx context.Context, spec AgentSpec) (AgentHandle, error)

	// Stop terminates the agent and releases substrate-managed resources.
	// Idempotent: stopping an already-stopped or unknown agent is a no-op that
	// returns nil, not ErrAgentNotFound.
	Stop(ctx context.Context, handle AgentHandle) error

	// Status returns the current lifecycle state of the agent. It returns
	// ErrAgentNotFound if the handle is unknown.
	Status(ctx context.Context, handle AgentHandle) (AgentStatus, error)

	// Exec runs a command inside the agent's execution context and returns its
	// exit code, stdout and stderr. Used for skill installation, health checks
	// and operator-initiated debugging. Every substrate must support it — even
	// local-process, where it shells out into the working directory. A non-zero
	// ExitCode is reported in the ExecResult, not as a Go error; err is non-nil
	// only when the command could not be run at all.
	Exec(ctx context.Context, handle AgentHandle, cmd []string) (ExecResult, error)
}

// Harness is the agent harness — the program that runs *inside* the agent and
// drives the coding loop. This is the "runtime / harness" axis of ADR-007, kept
// separate from the inference axis (which provider serves the tokens). Inference
// is configured per-agent via AgentSpec.Env (read by pi-ai) in v0; a typed
// inference field is an APPEND-ONLY future addition once the inference proxy
// lands (SDD-034i). There is deliberately no "claude" harness — see the package
// doc for the permanent ToS constraint.
type Harness string

// The agent harnesses iris supports. Pi is the v0 fleet runtime.
const (
	HarnessPi       Harness = "pi"       // earendil-works/pi, MIT — v0 fleet runtime
	HarnessOpenCode Harness = "opencode" // alternative interactive harness
	HarnessHermes   Harness = "hermes"   // autonomous / supervision plane (future)
)

// Validate reports whether h is one of the supported harnesses.
func (h Harness) Validate() error {
	switch h {
	case HarnessPi, HarnessOpenCode, HarnessHermes:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidHarness, string(h))
	}
}

// SubstrateType identifies where an AgentRuntime provisions agents. It is the
// selector a Factory uses to pick a substrate adapter (Factory itself ships
// with the substrate implementations, not in this contract package).
type SubstrateType string

// The substrates declared by ADR-002 Component 4.
const (
	SubstrateDocker     SubstrateType = "docker"     // single-host, primary consultoría delivery
	SubstrateKubernetes SubstrateType = "kubernetes" // multi-host / production (kubelab K3s)
	SubstrateLocal      SubstrateType = "local"      // dev only, no isolation
)

// AgentStatus is the lifecycle state of an agent. It is a closed enum, never a
// free-form string, so consumers can switch on it exhaustively.
type AgentStatus string

// The agent lifecycle states.
const (
	StatusStarting AgentStatus = "starting"
	StatusRunning  AgentStatus = "running"
	StatusStopping AgentStatus = "stopping"
	StatusStopped  AgentStatus = "stopped"
	StatusFailed   AgentStatus = "failed"
)

// String returns the status as a plain string.
func (s AgentStatus) String() string { return string(s) }

// Validate reports whether s is one of the lifecycle states.
func (s AgentStatus) Validate() error {
	switch s {
	case StatusStarting, StatusRunning, StatusStopping, StatusStopped, StatusFailed:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidStatus, string(s))
	}
}

// AgentSpec is the desired state of a single agent. (Name, TeamID) is the
// identity; the remaining fields are the mutable spec whose change is a drift
// (see SpecHash and ErrSpecDrift).
type AgentSpec struct {
	// Name is the agent's name, unique within a team.
	Name string
	// TeamID is the tenant boundary; every agent belongs to exactly one team.
	TeamID string
	// Image is the container image (Docker/K8s). Ignored by the local substrate.
	Image string
	// Command overrides the image's default entrypoint/CMD when non-nil. Leave
	// nil to run the image's baked-in agent entrypoint (the production norm for
	// iris agent images); set it for generic images or operator debugging. For
	// the local substrate it is the argv executed directly.
	Command []string
	// Harness is the agent harness that drives the loop (pi | opencode | hermes).
	Harness Harness
	// Env is the agent's environment, including the pi-ai inference config in v0.
	Env map[string]string
	// Resources is the requested compute. Advisory for the local substrate.
	Resources ResourceRequest
}

// ResourceRequest is a substrate-neutral compute request. Units are chosen so
// they map cleanly onto every substrate without leaking one substrate's vocab:
// CPUMillis → docker --cpus / k8s requests.cpu, MemoryBytes → docker -m /
// k8s requests.memory, GPUs → device count. Zero means "unset / substrate
// default". Advisory for the local substrate.
type ResourceRequest struct {
	CPUMillis   int   // 1000 = one core
	MemoryBytes int64 // bytes
	GPUs        int   // device count
}

// AgentHandle is an opaque, substrate-neutral reference to a running agent. The
// substrate-assigned ID is a container ID (Docker), a namespaced pod / Sandbox
// name (Kubernetes) or a PID (local). Carry it back into Status/Exec/Stop; do
// not parse it — its shape is the substrate's business.
type AgentHandle struct {
	ID        string        // substrate-assigned identifier
	Name      string        // echoes AgentSpec.Name
	TeamID    string        // echoes AgentSpec.TeamID
	Substrate SubstrateType // which substrate issued this handle
}

// ExecResult is the outcome of AgentRuntime.Exec. ExitCode is the command's own
// exit status (non-zero is not a Go error); Stdout/Stderr are its captured
// output.
type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// SpecHash returns a stable fingerprint over the *mutable* fields of the spec —
// Image, Command, Harness, Env and Resources. Name and TeamID are deliberately
// excluded: they are the agent's identity, not part of drift. Two specs with the
// same identity but a different SpecHash are a drift, and Start rejects the
// second with ErrSpecDrift. Env keys are sorted so the hash is order-independent.
//
// Design note (the one real choice here): "identity vs mutable spec". Changing
// Image/Command/Harness/Env/Resources mutates *this* agent (drift → explicit
// restart); changing Name/TeamID describes a *different* agent entirely. If you
// want, say, Resources to be hot-mutable (resize without restart) it would move
// out of this hash and behind a future Update() capability — but in v0 every
// field change is a drift.
func (s AgentSpec) SpecHash() string {
	parts := []string{
		fmt.Sprintf("image=%s", s.Image),
		fmt.Sprintf("command=%s", strings.Join(s.Command, "\x1f")),
		fmt.Sprintf("harness=%s", s.Harness),
		fmt.Sprintf("cpuMillis=%d", s.Resources.CPUMillis),
		fmt.Sprintf("memBytes=%d", s.Resources.MemoryBytes),
		fmt.Sprintf("gpus=%d", s.Resources.GPUs),
	}
	keys := make([]string, 0, len(s.Env))
	for k := range s.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("env:%s=%s", k, s.Env[k]))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}
