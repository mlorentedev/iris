---
id: adr-002-orchestrator-architecture
type: adr
status: accepted
created: "2026-05-16"
owner: manu
tags: [iris, orchestrator, architecture, consolidator]
---

# ADR-002 — iris Architecture (v0 canonical)

> **Status:** Accepted, 2026-05-16
> **Closes:** SDD-024 (master plan), SDD-026 (agent_crew analysis), SDD-033 (4-pattern extraction)
> **Depends on:** [adr-003-license-apache-2](./adr-003-license-apache-2.md) (license = Apache 2.0), [adr-001-frontend-htmx-go](./adr-001-frontend-htmx-go.md) (frontend = HTMX + Go templates)
> **Related:** agent-crew-analysis, candidate-nats-protocol-envelope, candidate-agent-runtime-capabilities, candidate-team-leader-workers, candidate-post-action-bindings, orchestrator-terminology, 2026-05-15-orchestrator-pivot

## Decision

`iris` is built from **four loosely-coupled components communicating exclusively over a NATS JetStream bus**, packaged as a **single Go binary plus a separate Python worker image**, deployed on **Docker or Kubernetes substrates** via a common runtime abstraction, served behind **an HTMX-driven operator console**, all under **Apache 2.0**.

```
┌──────────────────────────────────────────────────────────────────┐
│                  iris (single deployment unit)                  │
│                                                                  │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │           Go Motor (single binary)                      │   │
│   │  ─ HTTP API (cookie auth + CSRF, JSON+HTML)             │   │
│   │  ─ HTMX console (embed.FS templates)                    │   │
│   │  ─ Scheduler (team lifecycle, task dispatch)            │   │
│   │  ─ Webhook trigger + cron scheduler                     │   │
│   │  ─ PostAction dispatcher                                │   │
│   │  ─ Runtime adapter (Docker | K8s | local)               │   │
│   │  ─ Auth providers (noop | local-session | OIDC*)        │   │
│   └─────────────────────────────────────────────────────────┘   │
│                            ▲   ▼                                 │
│                    ╔═══════╧═══╧════════╗                        │
│                    ║   NATS JetStream   ║   ← typed envelope     │
│                    ║   (per-team subj)  ║   (see patterns)       │
│                    ╚═══════▲═══▲════════╝                        │
│                            │   │                                 │
│   ┌────────────────────────┴───┴───────────────────────────┐    │
│   │       Python Workers (one container per agent role)    │    │
│   │  ─ Hermes runtime adapter (Nous Research, MIT)         │    │
│   │  ─ OpenCode runtime adapter (peer interactive)         │    │
│   │  ─ Ollama runtime adapter (local on-demand)            │    │
│   │  ─ Skill loader + MCP client                           │    │
│   │  ─ Activity event emitter                              │    │
│   └────────────────────────────────────────────────────────┘    │
│                                                                  │
│   Shared infra (per substrate):                                  │
│   ─ Qdrant (vector store, when RAG enabled)                      │
│   ─ Ollama container (when local models enabled)                 │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
                              ▲   ▼  HTTP (optional)
                      ┌───────┴───┴────────┐
                      │  External systems  │
                      │  ─ n8n (D19)       │  ← bidirectional via
                      │  ─ Slack, CRM,     │     webhook/PostAction
                      │    PagerDuty, etc. │     (NOT bundled)
                      └────────────────────┘
```

(*OIDC declared but not implemented in v0 — see "What's NOT decided".)

## Scope

What this ADR covers:
- Canonical architectural decomposition: 4 components, their contracts, their boundaries.
- Deployment topology: single binary + worker image + optional shared infra.
- Cross-component communication: NATS subjects + envelope shape (refs candidate-nats-protocol-envelope).
- Substrate abstraction: how Docker, Kubernetes, and local-dev all satisfy the same runtime contract.
- Composition of the 4 extracted patterns into one coherent system.

What this ADR does NOT cover:
- Wire-level schema details (lives in the 4 pattern docs).
- License (decided in [adr-003-license-apache-2](./adr-003-license-apache-2.md)).
- Frontend specifics (decided in [adr-001-frontend-htmx-go](./adr-001-frontend-htmx-go.md)).
- Observability instrumentation (deferred to SDD-029 / future ADR-039).
- Specific Hermes / OpenCode / Ollama runtime adapter implementations (each gets its own follow-up ADR when built).

## Context

Today is 2026-05-16, the close of a multi-session reflection sprint that produced (in chronological order):
1. **The pivot decision** (2026-05-15, brainstorm): `ai-workloads-starter` reframes as a multi-agent orchestration platform (`iris`), blocking the existing kubelab roadmap.
2. **Scope clarification (D18, 2026-05-16):** target is freelance + corporate consultoría delivery, NOT SaaS open-core. See orchestrator-terminology.
3. **Prior-art analysis (agent-crew-analysis, 2026-05-16):** our own prior project agent_crew is the closest production-grade equivalent — same wedge, same substrate split. Its architecture and code are first-party and reusable in iris.
4. **License decision ([adr-003-license-apache-2](./adr-003-license-apache-2.md)):** Apache 2.0. Patterns extractable from AgentCrew with attribution; code is NOT.
5. **Four extracted patterns:** NATS protocol envelope, AgentRuntime + capabilities, team-leader-workers coordination, post-action bindings.
6. **Frontend decision ([adr-001-frontend-htmx-go](./adr-001-frontend-htmx-go.md)):** HTMX + Go templates. Single binary deploy.
7. **n8n coexistence (D19):** HTTP-protocol compatible bidirectionally; NOT bundled, NOT depended on.

This ADR consolidates all of the above into the architecture-of-record for iris's v0 implementation. It is the authoritative reference; downstream tasks and ADRs descend from it.

## The four components

### Component 1 — Go Motor (single binary)

The Go motor is the operational core. ONE binary, ONE process, ONE container. Responsibilities:

| Responsibility | Owns | Boundary |
|---|---|---|
| HTTP API + HTMX console | Operator-facing surface, cookie auth + CSRF, REST endpoints, partial HTML for HTMX swaps, SSE for activity stream | Does NOT execute agent code |
| Scheduler | Team lifecycle (create/deploy/stop), task dispatch to workers via NATS, schedule cron firing | Does NOT decide what an agent does — only when and where to run it |
| Trigger plane | Webhook inbound (`POST /webhook/trigger/:token`), schedule cron tick | Source of `TriggerOutcome` events feeding PostActions |
| PostAction dispatcher | Outbound HTTP per candidate-post-action-bindings, retry logic, concurrency limits, audit via `PostActionRun` table | Does NOT bundle integrations (n8n, Slack, etc. — all generic HTTP) |
| Runtime adapter | Translates `AgentSpec` to substrate operations per candidate-agent-runtime-capabilities | Does NOT contain substrate-specific business logic |
| Auth plane | Pluggable providers via `Provider` interface (noop, local-session-cookie, OIDC*) | Does NOT do authorization (RBAC) in v0 — that's a follow-up |
| Persistence | **`sqlc` (code-gen from SQL) + `golang-migrate` (schema migrations)** over SQLite for v0. Postgres adapter post-MVP if multi-tenant scale appears | Schema-owned tables only; workers don't touch the DB |

**Why sqlc + golang-migrate over GORM:** sqlc generates typed Go functions from SQL files at build time — compile-time errors when schema and queries drift, zero reflection at runtime, queries readable as plain SQL. GORM provides an ORM with runtime reflection — convenient but error-prone (schema-mismatch surfaces in production), slower, and harder to debug. Per CLAUDE.md's "stdlib > battle-tested libs > new dependencies" preference, sqlc is the right choice. `golang-migrate` handles schema versioning (the orthogonal problem sqlc deliberately doesn't solve). 2-3h ramp-up; pays back across the project's lifetime.

**What the motor does NOT do:**
- Execute agent code (workers do that).
- Speak to LLM providers directly (workers do that).
- Manage Qdrant or Ollama content (workers + shared infra do that).
- Render UIs other than HTMX-driven HTML (no SPA, no JSON-only API for browser use).

### Component 2 — Python Workers (per-agent containers)

Each agent in a Team is a Python worker container. Workers are stateless processes that:

| Responsibility | Owns | Boundary |
|---|---|---|
| Runtime execution | Drive Hermes (Nous Research) / OpenCode / Ollama via their respective adapters per the `Provider` enum (no `claude` — see "Why no `claude` provider" below) | Does NOT spawn or manage sibling workers |
| NATS subscription | Subscribe to its team's leader or worker subject per candidate-team-leader-workers | Does NOT subscribe to other teams' subjects (enforced at NATS connection-per-team level in v0; subject ACLs are future capability) |
| Skill loading | Load skill bundles from a configured path; install MCP servers per agent config | Does NOT modify iris's filesystem |
| Activity emission | Emit `activity_event` envelopes to `team.<name>.activity` on every semantic step (tool call, sub-agent dispatch, file read, model call) | Activity events are append-only descriptions, NOT control surfaces |
| Lifecycle signaling | Emit `container_validation` on startup, `skill_status` on skill load events, `mcp_status` on MCP connection events | Does NOT decide its own restart policy (motor owns that via runtime adapter) |

> **Superseded (2026-06-16) by [adr-009-worker-runtime-go](./adr-009-worker-runtime-go.md):** the worker is **Go**, not Python. The "ecosystem fit" rationale below assumed the worker embeds model SDKs and tool integrations; ADR-007 moved that responsibility into `pi` (a TypeScript dependency), so the worker is a thin pi-supervisor and Python's advantage no longer applies. The worker *contract* in this section (responsibilities, isolation, NATS-only boundary, idempotency, statelessness) is unchanged — only the implementation language is. "Python Workers" in this heading should be read as "fleet workers".

**Why Python (and not Go all the way through):** the AI agent ecosystem (LangChain, LlamaIndex, MCP SDK, model client libraries) has its center of gravity in Python. Forcing Go on workers would mean re-implementing or wrapping every model SDK + tool integration. Motor stays Go (operational correctness, single binary); workers are Python (ecosystem fit). The bus enforces the boundary.

**Why one container per agent role:** isolation. A misbehaving worker (model SDK leaks file handles, OOMs, hangs on a syscall) takes down ONE agent, not the team's leader or sibling workers. The substrate restarts the container per its policy.

### Why no `claude` provider (permanent architectural constraint)

The `Provider` enum is `hermes | opencode | ollama`. There is no `claude` value. This is **permanent**, not a current-scope decision. The reason is legal/policy:

Since April 2026, Anthropic's terms of service prohibit using Pro/Max subscriptions to provide agent capabilities to third parties via OAuth-based access. Claude Code authenticates via Manu's personal Anthropic subscription. Having iris invoke Claude Code on behalf of a consultoría client would constitute "reselling" subscription access — a clear ToS violation that would jeopardize both Manu's subscription and the client deployment.

If Anthropic ever offers an explicit "delegation" license, or if iris clients begin using their own Anthropic API keys directly (via the Anthropic API SDK, NOT via Claude Code CLI), the enum could be extended at that point with a new provider value (likely `anthropic-api`, not `claude` — to keep the distinction explicit). Until either of those changes, the enum is closed.

See orchestrator-terminology for the full naming convention and rationale.

### Component 3 — NATS JetStream Bus

The bus is the ONLY communication path between motor and workers. No HTTP, no shared filesystem, no database polling, no SSH. NATS-only. Envelope format and subject taxonomy are defined in candidate-nats-protocol-envelope.

**Topology in v0:**
- **Single NATS container per team in Docker mode** (isolation by deployment; one team = one NATS).
- **Single NATS cluster shared across teams in Kubernetes mode** (per-team connection isolation; one connection = one team).

**Multi-tenant isolation in v0:** the boundary is provided by **per-team NATS connection + in-process `team_id` discriminator on every API call**, NOT yet by NATS subject-based ACLs. Subject-based ACLs are a planned future capability (post-MVP) once a real multi-tenant deployment scenario appears. The v0 isolation is sufficient for consultoría delivery (each client = own iris deployment = own NATS) but is NOT sufficient for shared-NATS multi-tenant scenarios — that needs ACLs implemented and tested before claiming.

**Contractual constraint v0 (locked):** the FDE engagement contract template MUST prohibit shared multi-tenant deployments. Each customer = own iris deployment = own NATS = own SQLite. This converts the technical limitation into a contractual guarantee: even if a future motor bug mis-routes a `team_id`, the blast radius is bounded to the deployment of the customer who already owns the data. The clause stays in the template until NATS subject-based ACLs are implemented AND tested under adversarial conditions (penetration test or equivalent). Removing the clause requires an ADR amendment.

**Subject hierarchy** (recap):
```
team.<name>.leader      ← inbound to team's leader agent
team.<name>.activity    ← broadcast: activity events
team.<name>.worker.<w>  ← inbound to specific worker
team.<name>.system      ← iris lifecycle → all team components
```

**Why NATS JetStream over Kafka / RabbitMQ / Redis Streams:**
- Single binary deploy aligned with iris's wedge (no ZooKeeper, no broker cluster ceremony).
- JetStream provides at-least-once delivery + persistence sufficient for orchestration semantics.
- Subject hierarchy maps cleanly onto eventual subject-based authorization.
- Operational overhead measured in MB, not GB.

**Worker idempotency contract:** at-least-once means consumers MUST be idempotent. Workers dedupe by `MessageID`; iris's dedupe table is small (TTL = 1 hour, sufficient for retry windows).

### Component 4 — Runtime Substrate Adapter

Per candidate-agent-runtime-capabilities: a base `AgentRuntime` interface implemented by three substrates (Docker, Kubernetes, local-process), with optional capability sub-interfaces (`OllamaManager`, `QdrantManager`, `NetworkManager`, `SidecarManager`) that substrates opt into.

The runtime adapter lives in the Go motor (`internal/runtime/`), not in workers. Workers do not know whether they run on Docker or Kubernetes; their environment looks the same (NATS endpoint, model endpoints if applicable, working dir).

**Substrates supported in v0:**

| Substrate | Targeted use | Capabilities implemented |
|---|---|---|
| Docker | Single-host deployments — primary consultoría delivery model | base + Ollama + Qdrant + Network |
| Kubernetes | Multi-host / production scale | base + Ollama + Qdrant + Network + Sidecar |
| Local process | Developer machine — Manu's own dev workflow | base only (no Ollama/Qdrant local) |

## Composition — how the 4 components fit together

A user creates a Team via the HTMX console:

1. Operator opens console (HTMX) → `POST /api/teams` → motor writes Team + Agent rows.
2. Operator clicks Deploy → `POST /api/teams/:id/deploy` → motor's scheduler asks runtime adapter to provision: NATS (if Docker), agent containers, shared infra (Qdrant if needed).
3. Each worker container boots → connects to NATS → subscribes to its subject → emits `container_validation` envelope.
4. Motor receives `container_validation` events → marks Team `running` once all agents report healthy.
5. Operator types message in chat panel → `POST /api/teams/:id/chat` → motor publishes `user_message` envelope on `team.<name>.leader`.
6. Leader worker receives → executes its model call → may publish to `team.<name>.worker.<w>` to delegate → emits `activity_event` envelopes throughout.
7. Motor's SSE endpoint relays `activity_event`s to the operator's HTMX SSE-subscribed activity panel → operator sees real-time progress.
8. Leader emits `leader_response` envelope → motor relays to console → operator sees the team's reply.
9. As `TriggerOutcome` events fire (webhook complete, schedule complete), motor's PostAction dispatcher looks up bindings → fires outbound HTTP (including, optionally, to n8n).
10. Operator clicks Stop → motor's scheduler tears down via runtime adapter → containers die → NATS persists subject state for next deployment.

**The orchestration model is substitutable.** The leader worker is a black box (per candidate-team-leader-workers); whether it runs Hermes with a custom delegation loop, an Ollama-based reasoning loop, or a future LangGraph state machine — the console contract is unchanged. This is the design's most valuable property and the reason for the loose coupling.

## Inspiration footnote — Harness Engineering Lecture 08

The Harness Engineering analysis (see lesson "vocabulary donor vs architecture donor" in `90-lessons.md`) identified Lecture 08's component decomposition — **scheduler / verifier / handoff-reporter / progress-tracker** — as a useful contract surface for multi-agent systems. iris's 4-component decomposition is the same shape, mapped onto our substrate split:

| Harness Eng. concept | iris realization |
|---|---|
| Scheduler | Go motor's scheduler + runtime adapter (dispatch + provisioning) |
| Verifier | Go motor's PostAction outcome detection + workers' own validation steps |
| Handoff reporter | Workers emitting `activity_event` envelopes |
| Progress tracker | Motor's SSE relay + UI activity panel + future OTel instrumentation (SDD-029) |

This is not a forced fit — both designs converge on the same boundaries because the boundaries are real (centralized coordination vs distributed execution vs observability vs verification).

## What's DECIDED in v0

| Decision | Reference |
|---|---|
| License | [adr-003-license-apache-2](./adr-003-license-apache-2.md) (Apache 2.0) |
| Frontend stack | [adr-001-frontend-htmx-go](./adr-001-frontend-htmx-go.md) (HTMX + Go templates, single binary) |
| Provider abstraction | `hermes \| opencode \| ollama` — no `claude` (permanent, per ToS) |
| Bus | NATS JetStream, envelope per candidate-nats-protocol-envelope |
| Runtime abstraction | `AgentRuntime` + capabilities per candidate-agent-runtime-capabilities |
| Coordination model | Team→Leader→Workers, UI-oblivious to inter-agent handoff (per candidate-team-leader-workers) |
| Outbound integrations | Generic HTTP PostActions per candidate-post-action-bindings |
| Persistence (v0) | sqlc + golang-migrate over SQLite (boring tech, single-file DB matches single-binary wedge) |
| Auth (v0) | Cookie session + CSRF + bcrypt local provider; OIDC deferred |
| Deployment model | Single Go binary + one Python worker image + optional shared infra (Qdrant, Ollama) |
| n8n integration | Bidirectional HTTP, never bundled (D19, see [adr-001-frontend-htmx-go](./adr-001-frontend-htmx-go.md)) |
| Multi-tenant boundary (v0) | `team_id` discriminator on every row + per-team NATS connection (subject ACLs post-MVP) |

## What's NOT decided (deferred)

| Open question | Tracked in | Why deferred |
|---|---|---|
| Observability instrumentation (OTel traces/spans, cost attribution) | SDD-029 | MVP feature for console; needs design pass; not blocking v0 boot |
| NATS subject-based ACLs (multi-tenant in shared cluster) | None yet (post-MVP) | v0 per-team-connection isolation is sufficient for consultoría delivery |
| Postgres adapter for multi-tenant scale | None yet (post-MVP) | SQLite sufficient until ≥5 paying clients; premature optimization |
| OIDC implementation | None yet (post-MVP) | local-session covers freelance/consultoría delivery; enterprise auth gates triggered by procurement requirement |
| RBAC (authorization beyond authentication) | None yet (post-MVP) | Single-tenant-per-deployment in v0 obviates fine-grained roles |
| RAG pipeline depth (chunker, embedder, parser choices) | SDD-resurrect-as-needed | Qdrant is the chosen store; pipeline lives in workers; decisions arrive when a real RAG use case lands |
| Skill bundle distribution / marketplace | Implicit | Skills are loaded from a configured path in v0; central marketplace is a separate product question |
| Visual workflow builder | None — RESERVED for re-eval | Would trigger frontend re-decision (see [adr-001-frontend-htmx-go](./adr-001-frontend-htmx-go.md) re-evaluation triggers) |
| Hermes-specific runtime adapter design | Future ADR | Implementation detail; not architectural |
| OpenCode runtime adapter design | Future ADR | Same |
| Ollama runtime adapter design | Future ADR | Same |
| Fine-tuning architecture | Closed: `knowledge/11-tasks.md` SDD-025 wontfix 2026-05-16 | Out of scope; RAG-first |

## Consequences

**Positive:**
- The architecture is **fully derivable from 4 patterns + 2 ADRs**. A new contributor reads 6 short documents and has the complete picture.
- **Substrate-agnostic at the protocol layer.** Switching Docker→K8s is a deployment configuration change, not an architecture migration.
- **Substitutable orchestration strategy.** Leader is a black box; iris can adopt any agent-coordination model (CrewAI-style, LangGraph, custom Hermes loop) without UI or API changes.
- **First-party prior art.** agent_crew is our own project; its patterns and code are reusable in iris.
- **Wedge preserved end-to-end.** Single Go binary for motor + console, one Python worker image, single NATS container per team — operator deploys 2 things, not 7.
- **n8n integration latent and free.** No code commits needed beyond what's planned; capability available the moment a client asks.

**Negative / constraints:**
- **Workers in Python, motor in Go.** Polyglot — two language toolchains. Mitigated by Python being isolated in workers, motor + console + ops being Go.
- **NATS is the SPOF of cross-component communication.** Designed-for: JetStream provides persistence + replay; per-team NATS in Docker mode limits blast radius.
- **Leader is a per-team SPOF.** Restart-on-crash + thread replay are the mitigations; explicit failover is post-MVP.
- **Multi-tenant in v0 relies on per-team NATS connection isolation, NOT subject ACLs.** This is sufficient for "each client = own deployment" but is NOT a SaaS-grade boundary. Procurement audits that ask "what prevents cross-tenant message leakage" need the honest answer "currently, deployment isolation; ACLs are roadmap".
- **Single-binary deploy + embedded SQLite makes horizontal scaling non-trivial.** Acceptable for consultoría delivery (each client = own deployment); becomes a roadmap item if SaaS scenario ever appears (which D18 says it won't).

## Re-evaluation triggers

Revisit this ADR if any of the following becomes true:

- Pricing model shifts from consultoría delivery to SaaS multi-tenant. Architecture remains broadly valid but persistence (SQLite→Postgres), auth (cookie-session→OIDC), tenant isolation (NATS connections→subject ACLs), and observability (per-tenant cost) all need re-design.
- Substrate adoption demands a third substrate beyond Docker/K8s (Nomad, podman, systemd). Trivial extension via `AgentRuntime`; ADR doesn't need rewriting, just an implementation note.
- Frontend re-evaluation triggers from [adr-001-frontend-htmx-go](./adr-001-frontend-htmx-go.md) fire. This ADR's API surface remains unchanged; only the console layer changes.
- An external runtime appears that warrants direct integration as `Provider` value (e.g. a new autonomous-agent runtime that supersedes Hermes). Adding it is a new `Provider` enum value + adapter implementation; ADR notes the addition.
- The architecture's leader-as-SPOF becomes a real client complaint. Multi-leader voting model becomes an additive design — extend candidate-team-leader-workers and update this ADR's "What's decided" table.
- Anthropic's ToS changes to permit subscription delegation OR clients begin using their own Anthropic API keys. The `Provider` enum could be extended with an `anthropic-api` value at that point.

## Implementation roadmap (out-of-ADR, tracked on the bitácora board)

This ADR is the architecture-of-record. Implementation tasks belong in the project backlog:
- SDD-024 (orchestrator master plan) — closed as superseded by this ADR.
- New SDDs spawned for each component's v0 implementation (motor skeleton, worker skeleton, NATS protocol package, runtime adapter for Docker, HTMX console scaffold, first integration test) — see SDD-034a..f on the bitácora board (#55-59, #74).

Tracking belongs on the bitácora board (per ADR-018); this ADR does not enumerate work — only architecture.
