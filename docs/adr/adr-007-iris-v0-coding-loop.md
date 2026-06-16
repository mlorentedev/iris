---
id: adr-007-iris-v0-coding-loop
type: adr
status: active
created: "2026-06-13"
---

# ADR-007 — Execute iris v0: pi-driven, NATS-bussed, verify-gated coding loop

> **Status:** Accepted, 2026-06-13
> **Supersedes** an earlier deferral of the Go motor (un-defers it — see "Un-deferring the Go motor").
> **Executes / refines:** [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md) (architecture-of-record — unchanged; this ADR activates it and refines two points: runtime/inference axes + verification gate).
> **Depends on:** [adr-003-license-apache-2](./adr-003-license-apache-2.md) (Apache 2.0), [adr-001-frontend-htmx-go](./adr-001-frontend-htmx-go.md) (console deferred in v0).
> **Tracking:** mlorentedev/knowledge#93 (SDD-042 execution epic).
> **Related:** agent-crew-analysis, swe-af-analysis, orchestrator-terminology, candidate-nats-protocol-envelope, candidate-agent-runtime-capabilities, candidate-team-leader-workers, candidate-post-action-bindings.

## Decision

Build **iris v0** as an executable replica of the **agent_crew architecture** (Go motor + NATS JetStream bus + per-agent workers + PostAction triggers), with **three deliberate departures** that make it ours:

1. **`pi` is the fleet runtime** (not Claude-Code-via-sidecar). [earendil-works/pi](https://github.com/earendil-works/pi) (a.k.a. `badlogic/pi-mono`, Mario Zechner / Earendil Inc., **MIT**) is a headless-capable coding agent (TUI / print-JSON / RPC / SDK) whose `pi-ai` layer is a **unified LLM API over 20+ providers**. Inference for the fleet flows through `pi-ai` over **OpenRouter | NaN | Ollama-local** (all OpenAI-compatible → switchable by config). Claude Code never runs in the fleet (permanent ADR-002 ToS policy); pi's own Claude-OAuth path is excluded for fleet use by policy, not capability.
2. **A two-stage verification gate is mandatory** — the moat agent_crew structurally lacks. Coder emits a **typed self-check** in the PR (each acceptance criterion → evidence, with an anti-cheat rule: may not weaken or skip tests to go green); an **independent QA agent** (a second pi invocation) emits a typed verdict `FIX / APPROVE / BLOCK` (SWE-AF's synthesizer, reduced); then **human sign-off** (the variance control — never auto-merge).
3. **n8n coexists permanently** as the integration/trigger/notification layer (Slack / Telegram / Discord / external webhooks / cron), split from the Go motor by **strength**, not as a temporary placeholder.

This **un-defers the Go motor** (reversing an earlier deferral), ratified consciously on 2026-06-13: the team chose "full motor + NATS" over the minimal alternatives because reusing our own prior agent_crew architecture is *implementing already-extracted patterns*, not greenfield scaffolding — which neutralises the solitary-scaffold-stall risk the earlier deferral was guarding against.

v0 also **ships as a kubelab app**: the Go motor's embedded **HTMX operator console** (ADR-001, brought forward into v0) deployed on kubelab K3s behind Traefik IngressRoute (`iris.<env>.kubelab.live`) + **Authelia forwardAuth**, on the already-running PostgreSQL (pgvector) and Loki/Prometheus/Grafana. This makes iris "one more kubelab app" and closes the dogfood loop at the deployment layer. **Human sign-off happens in that Authelia-authenticated console** (channels remain for notification + fallback). The console is the *operator console* of the loop — runs in flight, activity stream (SSE), gate verdict, sign-off button — **not** a visual team/agent builder (that stays post-v0).

## Context

This ADR is the output of an architecture session (2026-06-13) that set out to define a "minimal iris" — the coding-agent crew that builds **imagesensortool / ImagingSuite** while dogfooding kubelab + dotfiles + iris in one loop. The session resolved the long-standing open question ("which runtime codes?") and, through the user's corrections, converged on a larger-than-minimal but bounded-and-solid v0.

Inputs (all verified in-session):
- **agent-crew-analysis** — agent_crew's operating model: Team→leader→NATS→**sidecar(Go)+Claude Code CLI**→tool calls; activity streamed back over NATS→WebSocket; Schedules/Webhooks/PostActions as triggers; GORM/SQLite, multi-tenant by `OrgID`. **Its fatal gap: zero verification** (trust + path allowlist only). First-party prior art (our own project) — its code is reusable in iris; the worker is replaced by pi (MIT) regardless, which does Claude Code's job better.
- **swe-af-analysis** — donates the missing process/governance layer: bounded nested control loops, **typed FIX/APPROVE/BLOCK verdict**, per-role model routing, anti-cheat CI constraint. Apache-2.0 (liftable).
- **[adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md)** — the architecture-of-record (4 components over NATS). Unchanged; this ADR executes it.
- **Earlier deferral** — the Go motor had been deferred "until real coordination friction appears." Un-deferred here.
- **pi.dev** (researched in-session) — the runtime discovery that collapses the "which runtime + which inference backend" question into a single MIT dependency.

State verified on disk (Phase A): `~/Projects/iris/` and `~/Projects/imagingsuite/` **do not exist yet** (repo creation is the act that un-defers); board `mlorentedev/knowledge` has the SDD-034a-i stream live (#74, #55-59, #75-77).

## Three planes (the conceptual model)

The session's central clarification — "one thing is my development, another is the tasks I delegate to agents":

| Plane | Who / what | Runtime | Inference | Product? |
|---|---|---|---|---|
| **DEV** — building iris | Manu | **Claude Code** (personal harness) | Anthropic sub | No — out of the enum, out of the product |
| **FLEET** — agents iris dispatches | pi workers | **pi** | **OpenRouter / NaN / Ollama-local** (via `pi-ai`) | **Yes — this *is* iris** |
| **SUPERVISION** — what to do / when to stop | v0 = **human** (dispatch + sign-off) → future = **hermes** | — | — | Differential layer (deferred to ADR-008 / #72) |

Claude Code never crosses from DEV into FLEET. This is the same boundary ADR-002 / orchestrator-terminology draw, now made explicit as a tri-plane model.

## Components (v0 — copy agent_crew + additions)

| Component | Role | Origin |
|---|---|---|
| **Go motor** (single binary) | HTTP API + scheduler + worker lifecycle + runtime adapter (Docker) + persistence (sqlc + golang-migrate over SQLite, per ADR-002) | copy `agent_crew_api` architecture |
| **NATS JetStream bus** | motor↔worker transport; typed envelope + subject taxonomy | candidate-nats-protocol-envelope (extracted pattern) |
| **pi workers** (headless) | the agents that code; subscribe to NATS; inference via `pi-ai` | replaces agent_crew's sidecar + Claude Code CLI |
| **n8n** | external triggers + PostActions + multichannel notify (Slack/Telegram/Discord) | our layer (agent_crew's is minimal); D19 |
| **Verification gate** | coder self-check + QA agent (pi) + human sign-off | SWE-AF (agent_crew has none) — **the moat** |
| **Operator console** (HTMX, embedded in motor) | runs in flight + activity SSE + gate verdict + **sign-off button** | ADR-001 (brought into v0); copy agent_crew's *page taxonomy* as reference, not its React |
| **kubelab K3s deployment** | iris as a kubelab app: Kustomize base+overlay, IngressRoute, Authelia forwardAuth, Postgres, observability | SDD-034g (#75) + iris 00-context integration requirements |

## The verification gate (the moat — detailed)

1. **CI green** — objective sensor (GitHub Actions on the PR).
2. **Coder self-check** — pi worker writes a typed block in the PR body: each issue acceptance criterion → evidence (test name / command output). **Anti-cheat rule (hard):** the worker may not delete, skip, weaken, or `xfail` tests to pass — enforced by the QA agent and by diff review.
3. **Independent QA agent** — a second pi invocation (distinct system prompt, no shared state) reviews diff + self-check + CI and emits a typed verdict: `FIX` (back to coder, bounded iterations) | `APPROVE` (to human) | `BLOCK` (escalate to human with reason).
4. **Human sign-off** — Manu approves in the **HTMX operator console** (authenticated by kubelab's **Authelia forwardAuth** — reuses the existing identity gate, no new auth) or via channel/GitHub → merge. **Never auto-merge** (hard rule; the variance control — expert sign-off).

## Runtime × Inference axes (refines ADR-002's enum)

pi confirms that ADR-002's flat `Provider` enum (`hermes | opencode | ollama`) conflated two orthogonal axes. ADR-007 separates them:

- **Runtime / harness axis:** `pi` (v0 fleet) · `opencode` (alternative) · `hermes` (autonomous / supervision, future).
- **Inference-provider axis (via `pi-ai`):** `OpenRouter` · `NaN` · `Ollama-local` · `Anthropic-API` (only if a client brings their own key) — **never Claude-Code-OAuth in the fleet**.

`ollama` moves from the runtime enum to the inference axis, where it always belonged. Switching Ollama↔OpenRouter, or pi↔opencode, becomes config behind the loop contract — not re-architecture. The intelligent inference proxy (cost/availability routing + PII redaction + budgeting) is **#77 (SDD-034i)**: the seam is placed here, full implementation deferred.

## The loop

```
issue (label `iris`)
  → n8n trigger (or webhook / cron)
  → Go motor dispatches a pi worker in a git worktree (NATS `user_message` envelope)
  → pi codes · inference via pi-ai (OpenRouter | NaN | Ollama) · emits `activity_event`s
  → opens PR + typed self-check (criterion → evidence, anti-cheat)
  → QA agent (pi) → verdict FIX / APPROVE / BLOCK
  → motor PostAction / n8n → notify verdict + PR to Telegram / Slack / Discord
  → human sign-off (channel or GitHub) → merge
```

`worktree = merge-conflict boundary` (SWE-AF), orthogonal to `container = security boundary` (when on K3s).

## n8n ↔ Go motor coexistence

Split by strength; both permanent. The Go motor does **not** replace n8n — it takes the responsibilities n8n does poorly:

| Responsibility | Owner | Why |
|---|---|---|
| Multichannel integration, external triggers, visual automation workflows | **n8n** (permanent) | exactly what it is for; rebuilding in Go would discard working capability |
| Worker lifecycle, typed bus, transactional state, low-level dispatch guarantees | **Go motor** | n8n does these poorly; build only here |

Replacement of one by the other happens only if strongly justified; default is coexistence.

## Relationship to agent_crew (what we copy / reimplement / ignore)

| agent_crew artefact | Verdict | Why |
|---|---|---|
| NATS envelope + subject taxonomy | **COPY (pattern)** | clean, license-free idea (candidate-nats-protocol-envelope) |
| `runtime` interface (Docker/K8s + capabilities) | **COPY (pattern)** | candidate-agent-runtime-capabilities |
| Data models (Team/Agent/Schedule/Webhook/PostAction), REST routes | **COPY (pattern)** | well-designed taxonomy; adopt as skeleton |
| Team→leader→workers coordination | **COPY (pattern)** | candidate-team-leader-workers — leader is a black box |
| `internal/claude/` subprocess manager | **REIMPLEMENT with pi** | 80%+ of their orchestration assumes Claude Code's process model; cloning the repo inherits coupling we'd have to rip out anyway. pi does this better, MIT |
| React UI | **IGNORE** | v0 UI = channels + GitHub; HTMX console is ADR-001 (deferred) |

**Net:** reuse our own prior architecture + pi (MIT) + add the verification gate it lacked. Apache 2.0.

## Un-deferring the Go motor

An earlier decision deferred the Go motor "until real coordination friction appears (scaffolding is commodity)." This ADR **un-defers it**, on these grounds, ratified by the maintainer 2026-06-13:
- Reusing our own prior agent_crew architecture is **implementing already-extracted patterns**, not greenfield scaffolding — the solitary-scaffold-stall risk does not apply.
- A **real forcing function now exists** (imagesensortool), the missing precondition.
- **pi (MIT)** collapses the worker + inference cost to a dependency, not a build.

The deferral of the **resident loop-mode supervisor** (the autonomous "what next / when to stop" layer) **still stands** — tracked as #72, gated on the human-in-the-loop v0 proving out first.

## What's decided in v0

| Decision | Note |
|---|---|
| Fleet runtime | **pi** (MIT) |
| Inference | `pi-ai` over OpenRouter / NaN / Ollama-local (switchable); never Claude in fleet |
| Architecture | agent_crew replica: Go motor + NATS + workers + PostActions |
| Verification gate | self-check + independent QA agent + human sign-off (the moat) |
| Integration / triggers / notify | **n8n** (permanent, coexists with motor) |
| Dispatch | issue (label `iris`) → n8n trigger → motor |
| Isolation | git worktree per issue |
| Persistence | sqlc + golang-migrate over SQLite (ADR-002) |
| License | Apache 2.0 (agent_crew is first-party — code reusable) |
| Forcing function | imagesensortool / ImagingSuite |
| Operator console | HTMX (embedded in motor, ADR-001) — runs/activity/verdict/sign-off; not a team-builder |
| Deployment | iris as a **kubelab app** on K3s: IngressRoute + Authelia forwardAuth + Postgres + observability (SDD-034g/#75) |
| Human sign-off | in the Authelia-authenticated console (or channel/GitHub) |
| Repo | `~/Projects/iris/` created in v0 (the un-defer act) |

## What's deferred (with trigger)

| Deferred | Trigger to build |
|---|---|
| Resident loop-mode supervisor (hermes autonomous) | v0 human-in-loop proven → ADR-008 / #72 |
| Intelligent inference proxy (PII / budget / cost routing) | cost/PII matters → #77 (SDD-034i) |
| HTMX **team-builder** (visual team/agent editor) | post-v0; v0 ships the *operator* console only |
| Multi-tenant subject ACLs, OIDC, RAG depth | per ADR-002 deferral table |
| opencode as 2nd fleet runtime / 2nd-executor benchmark | optional increment after ≥1 merged PR |

## v0 build order (anti-stall)

v0 grew (motor + NATS + pi + gate + console + K3s). To avoid the solitary-scaffold stall, build in this order — each step shippable:
1. **Loop core** — motor + NATS + one pi worker + gate (self-check + QA agent); sign-off via CLI/channel. Goal: **first merged PR on imagesensortool**.
2. **Product layer** — HTMX operator console + Authelia sign-off + kubelab K3s deploy. Goal: iris as a kubelab app.
3. **Hardening** — second pi worker (concurrency exercises NATS), inference switch to NaN/Ollama, n8n channel polish.

The UI is **ordered, not deferred**.

## Consequences

**Positive:**
- A single coherent v0 that is **solid in contract** (typed bus, verification gate, switchable inference, real channels, real isolation) without inventing anything — every component is a copied pattern or an MIT dependency.
- **The moat is in from day one**: provable agent labor (CI + self-check + QA verdict + sign-off) — the exact gap agent_crew and the closed-model incumbents leave open.
- **pi unifies runtime + inference** under one MIT dependency; DEV and FLEET share tooling, only the provider differs.
- **Dogfood loop closes**: kubelab (substrate) + dotfiles (harness) + iris (factory) validated by building imagesensortool.
- **iris's strengths are concrete**: verification gate, sovereign/cheap inference (pi-ai over NaN/Ollama), rich n8n integration.
- **Ships as product from v0**: an Authelia-gated operator console deployed as a kubelab app — the deployment-layer dogfood, and the surface that makes iris feel like a product, not a script.

**Negative / constraints:**
- **Un-defers the Go motor** — more upfront build than the human-dispatched skeleton; mitigated by copying extracted patterns + pi, and by the real forcing function.
- **Polyglot** (Go motor + pi/TS workers) — same trade-off ADR-002 already accepted.
- **Fleet model quality** (pi over NaN/Ollama) may trail Claude Code; the gate + bounded FIX loop absorb variance, and provider is switchable to OpenRouter when quality matters.
- **n8n becomes an operational dependency** of the loop (acceptable: already running on kubelab).

## Re-evaluation triggers

- **Third-party delivery (the ToS boundary):** before any client engagement, verify the fleet excludes every Claude path (provider ∈ {OpenRouter, NaN, Ollama, client-API-key}); reaffirm in the contract template (ADR-002 multi-tenant clause).
- **Coordination friction proven manageable without it** → reconsider NATS weight (unlikely once chosen, but logged).
- **Resident autonomy wanted** (remove human from dispatch) → ADR-008 / #72.
- **Fleet quality consistently fails the gate** → revisit runtime/provider mix or escalate human involvement.

## References

- [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md) · [adr-003-license-apache-2](./adr-003-license-apache-2.md) · [adr-001-frontend-htmx-go](./adr-001-frontend-htmx-go.md)
- agent-crew-analysis · swe-af-analysis · orchestrator-terminology
- pi: https://pi.dev/ · https://github.com/earendil-works/pi
- Tracking: mlorentedev/knowledge#93 (SDD-042) · ADR-008 candidate: #72 (SDD-041)
