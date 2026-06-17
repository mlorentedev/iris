---
id: adr-009-worker-runtime-go
type: adr
status: accepted
created: "2026-06-16"
---

# ADR-009 — iris fleet worker is Go, not Python

> **Status:** Accepted, 2026-06-16
> **Supersedes:** [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md) §"Component 2 — Python Workers" / §"Why Python (and not Go all the way through)". The worker's *responsibilities, isolation model, NATS-only boundary, idempotency, and statelessness are unchanged* — only the implementation language changes.
> **Refines:** [adr-007-iris-v0-coding-loop](./adr-007-iris-v0-coding-loop.md) (the pi-as-fleet-runtime decision is what invalidates ADR-002's Python rationale).
> **Tracking:** mlorentedev/iris#5 (SDD-034d).

## Decision

The iris fleet worker is implemented in **Go**, in this repository, sharing the `internal/protocol` package with the motor. It drives **pi** ([earendil-works/pi](https://github.com/earendil-works/pi), MIT, a TypeScript coding-agent CLI) as a **subprocess** (CLI / print-JSON / RPC interface), not as an in-process library. Python is dropped as the worker language.

## Context

ADR-002 specified Python workers. Its sole rationale was *ecosystem fit*: "the AI agent ecosystem (LangChain, LlamaIndex, MCP SDK, model client libraries) has its center of gravity in Python." That argument is load-bearing **only if the worker embeds the agent loop** — i.e., calls model SDKs and wires tool integrations itself.

Two facts, both established after ADR-002, remove that premise:

1. **ADR-007 moved the agent loop out of the worker and into `pi`.** pi owns the coding loop and, via `pi-ai`, owns inference over OpenRouter / NaN / Ollama. ADR-007: "pi collapses the worker + inference cost to a **dependency, not a build**." The iris worker is therefore a **thin supervisor** — NATS subscribe, dispatch pi in a git worktree, stream `activity_event` — not an agent-logic host. Python's ecosystem advantage is unused.
2. **pi is TypeScript, not Python.** A Python worker cannot embed pi via its SDK; it could only `exec` the pi CLI as a subprocess — exactly what a Go worker does — but without sharing the motor's language and without reuse of first-party Go prior art.

So Python is **dominated**: it shares a language with neither the motor (Go) nor the harness (pi, TS), while still paying the cross-language tax (a Pydantic envelope mirror + the `internal/protocol/testdata/*.json` fixtures maintained as a cross-language oracle + a parallel Python toolchain/CI).

## Decision drivers (why Go)

- **One language with the motor.** The worker imports `internal/protocol` directly. The envelope stops being a contract to *replicate* and becomes a *shared dependency* — eliminating the Pydantic mirror, the cross-language oracle role of `testdata/*.json`, and a second toolchain/CI.
- **First-party reuse as code, not patterns.** `agent_crew_api`'s Go sidecar (`cmd/sidecar`: the claim → exec → emit loop, NATS lifecycle, the `entrypoint.sh` workspace-UID fix) is reusable directly.
- **One toolchain, strong concurrency.** The worker compiles to a single static Go binary in the same `go.mod`/CI as the motor, with the goroutine model that fits the NATS + subprocess + heartbeat workload. (Caveat — see Consequences: the worker *runtime image* is **not** distroless like the motor's, because pi is npm-only and must run on a Node base. That Node requirement is identical for any worker language, so it does not bear on this decision.)
- **Industry shape.** The agent *loop* lives in Python where the framework embeds it (OpenHands, SWE-agent); the *control plane* lives in Go (Temporal, Daytona, agent-sandbox); the *harness* lives in TS (pi, Claude Code, Codex CLI). iris extracted the loop into pi, so its worker is control-plane-adjacent → Go.

## Consequences

- The `internal/protocol` package doc comment ("FROZEN wire contract between the Go motor and the **Python workers**") must be updated to reflect a Go worker. The `testdata/*.json` fixtures remain valuable as protocol golden tests, but no longer exist to bridge two languages.
- **Incidental "Python workers" mentions to sweep** (not rewritten here — a focused doc-reconciliation task): `adr-001-frontend-htmx-go` §5 ("Go motor + Python workers" — note: Go workers *strengthen* ADR-001's "fewer languages" argument), `adr-003-license-apache-2` (license scope bullet), and the `internal/protocol/messages.go` package doc. None change a decision; all should read "Go / fleet workers" once the 034d implementation lands.
- The worker lives in-repo (`cmd/worker/` + `internal/worker/`), one Go module, one CI — no `workers/` Python subtree.
- **The worker runtime image is a Node base, not distroless.** pi is npm-only (`@earendil-works/pi-coding-agent`; no standalone binary), so the worker image is multi-stage: Go-build → Node-runtime with pi installed `--ignore-scripts` and pinned, plus the Go binary copied in. pi version upgrades are dependency bumps (pinned + tested). The Go worker drives pi via its `--mode json` (structured stdout events) or `--mode rpc` (JSON-RPC) interface, with the subprocess `cwd` set to the per-job git worktree.
- pi is driven over its CLI / print-JSON / RPC surface (subprocess). In-process pi extensibility (custom tools as TS functions, native event streaming via the SDK) is **not** available to a Go worker; if that becomes a hard requirement, revisit a TypeScript worker (see Alternatives) — which would reintroduce a language split.
- No change to ADR-002's worker *contract*: one container per agent role, NATS-only boundary, dedupe by `MessageID`, stateless, motor-owned restart policy.

## Alternatives considered

- **Keep Python (ADR-002 status quo).** Rejected: its only rationale (AI ecosystem) is moot post-pi; cannot embed pi (TS); pays the mirror + oracle + second-toolchain tax; no `agent_crew_api` reuse. Worst of the three on every axis that now matters.
- **TypeScript worker embedding the pi SDK.** Viable and gives the tightest pi integration (in-process tools, native event streaming, pi's "aggressive extensibility" in its own language). Rejected for v0: a third language alongside the Go motor, an envelope re-mirror in zod, a Node runtime image, and zero `agent_crew_api` reuse. Deferred unless deep in-process pi extensibility becomes a hard requirement — at which point this ADR should be revisited rather than worked around.
- **Architecture session.** Considered; the decision was clear enough from the ADR-002↔007 contradiction + pi being TS to record directly. This ADR is the persistence of that decision.
