// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package runtime

import "context"

// QdrantManager is the capability of provisioning a substrate-managed Qdrant
// vector store for RAG features. A substrate opts in by implementing it;
// consumers discover it via type assertion and MUST handle the not-implemented
// case with an agent-oriented (ERROR/WHY/FIX) error.
//
// Implemented by: Docker (separate container), Kubernetes (StatefulSet). NOT
// implemented by local-process (use an external Qdrant in dev).
type QdrantManager interface {
	// EnsureQdrant provisions the vector store if absent and returns its
	// endpoint once ready. It is idempotent.
	EnsureQdrant(ctx context.Context) (QdrantEndpoint, error)
	// QdrantCollections lists the collections present on the store.
	QdrantCollections(ctx context.Context) ([]string, error)
}

// QdrantEndpoint addresses a provisioned Qdrant instance. Both the HTTP (REST)
// and gRPC ports are reported so callers pick the protocol they need; the host
// is reachable from within the agent's network.
type QdrantEndpoint struct {
	Host     string
	HTTPPort int // Qdrant REST API (default 6333)
	GRPCPort int // Qdrant gRPC API (default 6334)
}
