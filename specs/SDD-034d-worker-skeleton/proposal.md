---
id: "SDD-034d-worker-skeleton"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-06-16"
issue: "iris#5"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# SDD-034d: Worker skeleton

> **Naming**: file lives at `<repo>/specs/<feature-id>/proposal.md`. `<feature-id>` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #5: SDD-034d: iris Go worker skeleton + pi CLI driver -->

The iris motor (Go) can already schedule agents and route NATS, but **nothing executes work yet** — there is no fleet worker, so the system is inert end-to-end (the umbrella smoke-test's `worker-pi` is a healthcheck-only stub). SDD-034d ships the **first real worker**: a Go process that subscribes to its `team.<team>.worker.<name>` subject, drives a pi coding session in a git worktree, and emits `activity_event` envelopes back on `team.<team>.activity`. Without it, the motor + bus + runtime adapters built in SDD-034a–c have nothing to orchestrate, and the v0 coding loop (ADR-007) cannot close its first merged PR on imagesensortool.

## What

Three concrete outputs that did not exist before this PR:

1. A new `cmd/worker` binary that, given `NATS_URL` + `TEAM` + `WORKER_NAME` + inference env, connects to the bus, subscribes to `team.<team>.worker.<name>`, and on a `user_message` envelope decodes it with `internal/protocol` (shared package, no mirror) and launches **pi** (via its print-JSON CLI) in an isolated git worktree.
2. The worker emits typed telemetry back: `container_validation` on startup and one `activity_event` per pi semantic step (tool call / file op / model call) on `team.<team>.activity`.
3. The `worker-pi` service in `compose.dev.yml` goes from healthcheck-only stub to the real binary; umbrella smoke-test checks #4 (worker running) and #5 (NATS pub/sub roundtrip) now pass against a real worker.

## Out of scope

Things this PR explicitly does NOT include. Forces a sharp boundary and prevents scope creep.

- **The two-stage verification gate** (coder typed self-check + independent QA agent + human sign-off) — ADR-007's moat, its own SDD. The skeleton only proves a worker drives pi and reports.
- **Leader→worker multi-agent delegation** / team coordination (candidate-team-leader-workers) — this PR has one worker on its own subject; the leader-as-black-box is later.
- **Skill-bundle + MCP-server loading** per agent config (ADR-002 Component 2), and **inference-provider switching** beyond passing env through to pi-ai (that is SDD-034i, the inference proxy).
- **Multiple concurrent jobs per worker (item E).** Each worker processes one job at a time — a *permanent* per-agent-container property (ADR-002 isolation), not a skeleton limitation. Fleet concurrency arrives by the motor dispatching *more* worker containers (ADR-007 Hardening: "second pi worker exercises NATS"), each still single-job.

## Risks / open questions

Failure modes, dependencies, and unknowns to clarify before implementation. If any item here is unresolved, do not move to `tasks.md` yet.

- **[BLOCKER — RESOLVED 2026-06-16] pi's headless invocation contract.** Pinned from pi `docs/sdk.md` + README: the skeleton drives `pi --mode json` (single-shot prompt → structured stdout event stream → exit), with `--mode rpc` (JSON-RPC over stdin/stdout) as the richer evolution. Subprocess `cwd` = the per-job git worktree; prompt via `-p`/stdin; provider/model via env (`pi-ai` OpenAI-compatible base — never `ANTHROPIC_API_KEY`) + `--model`. Event vocabulary: `agent_start`/`agent_end`, `tool_execution_start`/`tool_execution_end`, `message_update`, `turn_end`. _Residual:_ pi exit codes are undocumented — confirm empirically in Task 1.
- **[BLOCKER for the image — RESOLVED 2026-06-16] pi distribution in the worker container.** pi is npm-only (no standalone binary), so the worker image is multi-stage Go-build → **Node base** with pi installed `--ignore-scripts` and pinned + the Go binary copied in (recorded in ADR-009). The Node requirement is identical for any worker language, so it does not affect the Go decision. Dev loop uses a host-installed pi meanwhile.
- **[Resolve before code] pi event stream → `activity_event` mapping.** Translating pi's own event stream into the `activity_event` payload (`internal/protocol/payloads.go`, a FROZEN surface) may surface a protocol gap — extending a FROZEN surface needs care / an ADR.
- **Secondary:** idempotency/dedup by `MessageID` (ADR-002 contract, TTL 1h, item D) — **confirmed deferred** from the skeleton (own follow-up); git worktree lifecycle is now in scope (item C: created per job, removed on completion).

## Acceptance criteria

Observable outcomes. Each must be testable.

- [x] **Protocol roundtrip (shared Go package):** a `user_message` envelope published to `team.<team>.worker.<name>` is decoded via `internal/protocol` and dispatched — no mirror. (`internal/worker` test) — `TestDispatch*`
- [x] **pi driven in a worktree:** the worker spawns `pi --mode json` with `cwd` = an isolated git worktree and a prompt from the envelope, consumes its event stream to `turn_end`/`agent_end`, and **removes the worktree on completion** (no leaked worktrees — item C). (integration test against a **fake-pi** emitting canned JSON events — no real inference) — `TestDriverStreamsEventsToAgentEnd`, `TestWorktreeLifecycle`, `TestHandleHappyPath`
- [x] **Activity emission:** one `activity_event` on `team.<team>.activity` per pi `tool_execution_*` event; `container_validation` on startup. (test subscriber asserts envelopes) — `TestActivityFor*`, `TestHandleHappyPath`, `TestEmitStartup`
- [x] **Graceful shutdown (item A):** on SIGTERM the worker terminates the in-flight pi subprocess and drains within a bounded timeout — no orphaned pi processes. (test sends SIGTERM mid-job, asserts the pi child is reaped) — `TestDriverCancelReapsPi` + `cmd/worker` `signal.NotifyContext`
- [x] **pi failure surfaced, not swallowed (item B):** when pi exits non-zero or crashes, the worker emits a failure `activity_event` rather than dropping it silently. (fake-pi exits 1 → test asserts a failure envelope) — `TestHandleFailurePath`
- [x] **Smoke-test green:** umbrella `make smoke-test` checks #4 (worker-pi running) and #5 (NATS roundtrip) pass with the real binary in `compose.dev.yml`. (`make smoke-test` exit 0) — verified locally: `[smoke] 5/5 checks passed`, worker-pi healthy on `iris-worker:dev`

## References

- Tracking issue: `mlorentedev/iris#5` (work-gate)
- ADRs: [adr-009-worker-runtime-go](../../docs/adr/adr-009-worker-runtime-go.md) (worker is Go), [adr-007-iris-v0-coding-loop](../../docs/adr/adr-007-iris-v0-coding-loop.md) (pi runtime + loop), [adr-002-orchestrator-architecture](../../docs/adr/adr-002-orchestrator-architecture.md) §"Component 2" (worker contract)
- Frozen contracts consumed: `internal/protocol/{messages,subjects,payloads}.go` + `internal/protocol/testdata/` fixtures; `internal/runtime/runtime.go` (motor dispatches the worker container)
- Prior art (reuse as code): `agent_crew_api` Go sidecar (`cmd/sidecar` claim→exec→emit loop)
- pi: [docs/sdk.md](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/sdk.md) (`--mode json` / `--mode rpc`, event stream), README (npm distribution)
- Patterns: candidate-nats-protocol-envelope, candidate-team-leader-workers, candidate-agent-runtime-capabilities
