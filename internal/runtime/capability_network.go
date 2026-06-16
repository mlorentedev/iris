// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package runtime

import "context"

// NetworkManager is the capability of giving a team its own isolated network
// boundary and attaching agents to it. A substrate opts in by implementing it;
// consumers discover it via type assertion and MUST handle the not-implemented
// case with an agent-oriented (ERROR/WHY/FIX) error.
//
// Implemented by: Docker (user-defined bridge network per team), Kubernetes
// (NetworkPolicy scoped to the team). NOT implemented by local-process (host
// networking, no isolation).
type NetworkManager interface {
	// EnsureTeamNetwork creates the team's isolated network if absent. It is
	// idempotent.
	EnsureTeamNetwork(ctx context.Context, teamID string) error
	// AttachToTeamNetwork connects a running agent to its team's network.
	AttachToTeamNetwork(ctx context.Context, handle AgentHandle, teamID string) error
}
