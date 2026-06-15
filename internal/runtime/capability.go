// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package runtime

import "fmt"

// Capability discovery is plain type assertion: a runtime advertises a
// capability by implementing its sub-interface, and consumers detect it at the
// call site. The As* helpers below wrap that assertion so the FROZEN rule
// "every no-capability failure is an agent-oriented ERROR/WHY/FIX error" is
// encoded once instead of copy-pasted at every call site. They add no surface
// to the base interface — adding an accessor for a new capability later is an
// APPEND-ONLY change.

// MissingCapabilityError builds the standard agent-oriented error returned when
// a runtime does not implement a required capability. capability is the missing
// sub-interface's name; fix is the deployment change that would provide it.
func MissingCapabilityError(rt AgentRuntime, capability, fix string) error {
	return fmt.Errorf(
		"ERROR: capability %q unavailable in this deployment\n"+
			"WHY:   runtime %T does not implement the %s capability\n"+
			"FIX:   %s",
		capability, rt, capability, fix)
}

// AsOllamaManager returns rt as an OllamaManager, or a MissingCapabilityError.
func AsOllamaManager(rt AgentRuntime) (OllamaManager, error) {
	if m, ok := rt.(OllamaManager); ok {
		return m, nil
	}
	return nil, MissingCapabilityError(rt, "OllamaManager",
		"deploy iris on a substrate that provisions Ollama (Docker or Kubernetes), or disable features that require substrate-managed local models")
}

// AsQdrantManager returns rt as a QdrantManager, or a MissingCapabilityError.
func AsQdrantManager(rt AgentRuntime) (QdrantManager, error) {
	if m, ok := rt.(QdrantManager); ok {
		return m, nil
	}
	return nil, MissingCapabilityError(rt, "QdrantManager",
		"deploy iris on a substrate that provisions Qdrant (Docker or Kubernetes), or disable features that require RAG")
}

// AsNetworkManager returns rt as a NetworkManager, or a MissingCapabilityError.
func AsNetworkManager(rt AgentRuntime) (NetworkManager, error) {
	if m, ok := rt.(NetworkManager); ok {
		return m, nil
	}
	return nil, MissingCapabilityError(rt, "NetworkManager",
		"deploy iris on a substrate with per-team network isolation (Docker or Kubernetes); local-process uses host networking")
}

// AsSidecarManager returns rt as a SidecarManager, or a MissingCapabilityError.
func AsSidecarManager(rt AgentRuntime) (SidecarManager, error) {
	if m, ok := rt.(SidecarManager); ok {
		return m, nil
	}
	return nil, MissingCapabilityError(rt, "SidecarManager",
		"deploy iris on Kubernetes (sidecars need a shared Pod); Docker and local-process run one container per agent")
}
