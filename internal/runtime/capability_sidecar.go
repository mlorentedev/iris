// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package runtime

import "context"

// SidecarManager is the capability of co-locating an auxiliary container
// alongside a running agent, sharing its execution context. A substrate opts in
// by implementing it; consumers discover it via type assertion and MUST handle
// the not-implemented case with an agent-oriented (ERROR/WHY/FIX) error.
//
// Implemented by: Kubernetes (a second container in the same Pod). NOT
// implemented by Docker (one container per agent) or local-process (no
// isolation). This is the one capability that is K8s-only by nature.
type SidecarManager interface {
	// DeploySidecar attaches the sidecar to the agent referenced by handle.
	DeploySidecar(ctx context.Context, handle AgentHandle, sidecar SidecarSpec) error
}

// SidecarSpec describes an auxiliary container to co-locate with an agent.
type SidecarSpec struct {
	Name      string
	Image     string
	Env       map[string]string
	Resources ResourceRequest
}
