// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package runtime

import "context"

// OllamaManager is the capability of provisioning a substrate-managed Ollama
// inference server for local models. A substrate opts in by implementing it;
// consumers discover it via type assertion on an AgentRuntime and MUST handle
// the not-implemented case with an agent-oriented (ERROR/WHY/FIX) error.
//
// Implemented by: Docker (separate container), Kubernetes (Deployment in the
// shared-infra namespace). NOT implemented by local-process (Ollama is not
// orchestrator-managed in dev). Note this capability is orthogonal to the
// RuntimeType harness axis — "ollama" is an inference backend, never a runtime.
type OllamaManager interface {
	// EnsureOllama makes the named model available, pulling it if necessary, and
	// returns once it is ready to serve. It is idempotent.
	EnsureOllama(ctx context.Context, model string) error
	// ListOllamaModels returns the models currently available on the
	// substrate-managed Ollama server.
	ListOllamaModels(ctx context.Context) ([]string, error)
}
