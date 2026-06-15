// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

// Package runtime defines iris's substrate-neutral agent runtime contract: the
// FROZEN AgentRuntime base interface plus a set of APPEND-ONLY capability
// sub-interfaces. The motor's scheduler manages agents through this contract
// without knowing whether they run on Docker, Kubernetes or as local processes;
// the substrate adapters that implement it live in sub-packages
// (internal/runtime/docker, .../k8s, .../local) per ADR-002 Component 4.
//
// # Base vs capabilities
//
// AgentRuntime is the small, mandatory base every substrate satisfies:
// Start/Stop/Status/Exec — the universal agent lifecycle. Substrate-specific
// extras (Ollama and Qdrant provisioning, per-team networking, sidecars) are
// optional capability sub-interfaces — OllamaManager, QdrantManager,
// NetworkManager, SidecarManager — that a substrate opts into by implementing.
// Consumers discover a capability via type assertion at the call site (use the
// As* helpers in capability.go), never via a Capabilities() method: the type
// system, not a runtime lookup, is what proves a capability is present.
//
// Adding a capability is non-breaking (existing runtimes ignore it). Removing
// one is a breaking change handled by deprecation. The base interface is FROZEN:
// changing it requires an ADR plus a deprecation window.
//
// # Substrate neutrality (ADR-005)
//
// No type or method here may exist only to suit one substrate's shape. iris
// adopts kubernetes-sigs/agent-sandbox as its K8s substrate (OQ-1), but the
// fallback iris-native K8s adapter must stay implementable within two weeks —
// which is only true while this contract leaks nothing agent-sandbox-specific.
// The same AgentSpec/AgentHandle serve a Docker container, a K8s Sandbox CR and
// a local PID alike; ResourceRequest uses neutral units (CPUMillis, MemoryBytes,
// GPUs) that map onto every substrate.
//
// # Two axes: runtime/harness vs inference (ADR-007)
//
// ADR-007 split ADR-002's single Provider enum into two orthogonal axes. This
// package models the runtime/harness axis as Harness (pi | opencode | hermes) —
// the program that drives the coding loop inside the agent. The inference axis
// (which provider serves the tokens — OpenRouter | NaN | Ollama | Anthropic-API,
// via pi-ai) is configured per-agent through AgentSpec.Env in v0
// and is deliberately NOT a typed field yet: the inference proxy that owns that
// axis is deferred to SDD-034i, and the APPEND-ONLY rule lets a typed Inference
// field be added then without breaking this contract.
//
// Note OllamaManager is a *capability* (provision an Ollama server), wholly
// distinct from the now-retired "ollama" provider value: a substrate can serve
// local models regardless of which harness an agent runs.
//
// # No "claude" runtime (permanent constraint)
//
// RuntimeType has no "claude" value, by policy not omission. Since April 2026
// Anthropic's terms prohibit providing agent capabilities to third parties via
// OAuth subscription access; having iris drive Claude Code on a client's behalf
// would be reselling subscription access. If clients later bring their own
// Anthropic API key, the inference axis (not the runtime axis) gains an
// "anthropic-api" value — never "claude". See the candidate-agent-runtime-
// capabilities pattern and ADR-002.
//
// # Idempotency and spec drift
//
// Start is idempotent for an agent's (Name, TeamID) identity, but only while the
// spec is unchanged: a second Start with a mutated spec returns ErrSpecDrift
// rather than silently re-creating the agent. Silent re-create was rejected as a
// production footgun (operators ran stale images for days). See AgentSpec.SpecHash
// for exactly which fields constitute drift.
package runtime
