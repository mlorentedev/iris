---
id: adr-008-iris-product-factory
type: adr
status: accepted
created: "2026-06-14"
---

# ADR-008 — iris is the product: a three-layer agent-and-product factory (Go synthesis)

> **Status:** Accepted, 2026-06-14
> **Reframes:** [adr-007-iris-v0-coding-loop](./adr-007-iris-v0-coding-loop.md) — the verification gate drops from "the moat" to **one component** of the process layer. ADR-007's bottom-up build order survives; its framing does not.
> **Amends:** [adr-006-pivot-ai-native-engineering-studio](./adr-006-pivot-ai-native-engineering-studio.md) §4 — iris re-scoped from "internal machine, product deferred to Phase 3" to **iris IS the product**, dogfooded by building its first deliverable (imagesensortool).
> **Absorbs:** the ADR-008-candidate *resident loop-mode supervisor* (mlorentedev/knowledge#72 / SDD-041) → it is now **layer 3** (orchestration) of this synthesis, not a separate ADR.
> **Builds on:** [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md) (4-component Go architecture), [adr-003-license-apache-2](./adr-003-license-apache-2.md) (Apache-2.0).
> **References audited (Regla del 3, ADR-015):** agent-crew-analysis (deep code, vault), swe-af-analysis (design, vault), Paperclip analysis (paperclip.ing — deep-dive in progress, vault-pending).

## Context

An architecture session (2026-06-14) was triggered by discovering **Paperclip** (paperclip.ing, MIT, ~70k stars): an open-source control plane for "autonomous AI companies" that orchestrates agents as an org-chart toward a business goal — *"manage goals, not pull requests"*, and explicitly *"not a code review tool"*.

This forced two corrections the maintainer made in-session:

1. **The thesis is the factory, not the gate.** ADR-007 had framed iris around a verification-gated coding loop with the gate as "the moat" — a framing inherited from comparing iris against agent_crew as a rival. But iris's thesis (ADR-006) was always the **agent-and-product factory**: it "sells verified engineering outcomes produced by an in-house agent factory". The verification gate is **one component** (variance control), not the centre.
2. **iris IS the product** (the session's "option B"). Not internal machinery whose product is deferred to ADR-006's Phase 3 — the machine itself is the product, and its first validation is **dogfood-first**: building imagesensortool with it. Product and internal-machinery are not opposed; it is the YC thesis ADR-006 itself cites ("use the software yourself and sell the finished product").

The Go motor already exists and boots (Task 1 plumbing + Task 2 skeleton: chi + /healthz + /readyz + slog + graceful shutdown), so this ADR redirects an in-flight build, not a greenfield one.

## Decision

iris is **the product**: a sovereign **agent-and-product factory**, built as a **single Go binary** that **synthesizes the best of three audited references** into three composable layers, **dogfood-first** by building imagesensortool. Licences are no constraint (the synthesis is reimplemented in Go — patterns, not copied code — which is licence-agnostic by construction).

The differentiator is **not any single component** — the gate exists in swe-af, the meta-orchestration in Paperclip, the runtime in agent_crew. What none of them has is **the three together in one sovereign Go binary** with own-infrastructure inference (NaN/Ollama, never Claude in the fleet) and on-prem EU deployment. **The moat is the synthesis.**

### The three layers

| Layer | Donor reference | What is rescued |
|---|---|---|
| **1 · Runtime / infra** | agent_crew (AGPL → patterns only) | typed NATS envelope + subject taxonomy · `AgentRuntime` (Docker\|K8s + capability sub-interfaces) · **team→leader→workers (leader = black box)** · post-actions/webhooks/schedules · multi-tenant `OrgID` · capability-injection-at-spawn |
| **2 · Process / gate** | swe-af (Apache → code liftable) | bounded nested loops (coder→advisor→replanner with budgets) · **typed verdict FIX/APPROVE/BLOCK** · worktree = merge-conflict boundary · per-role model routing · checkpoint/resume · Postgres lease-based durable queue · anti-cheat CI |
| **3 · Orchestration / meta** | Paperclip (MIT → freely copyable) | **goal-ancestry** (business goal → tasks) · heartbeats (agent wake/schedule) · budgets & cost governance · org-chart / role hierarchy · approvals · workspaces · plugins · company portability |

The keystone that makes the synthesis cohere: **agent_crew's "leader = black box"** contract (layer 1) lets the meta-orchestration (layer 3, Paperclip-derived) plug in *as the leader's strategy* without breaking the runtime or the UI contract.

## Options considered

| Option | C1 sovereign | C2 no-fork | C3 dogfood | C4 synthesis | C5 anti-stall |
|---|---|---|---|---|---|
| **1 · Monolithic Go synthesis** (3 layers reimplemented) | ✓ | ✓ | ✓ | ✓ | ✓ |
| 2 · Go core + integrate Paperclip (Node) for layer 3 | gap (Node dep) | gap | ✓ | partial | ✓ |
| 3 · Fork Paperclip + Go gate bolt-on | gap (Node base) | ✗ | ✓ | gap | gap |

**Selected: Option 1.** Option 3 was rejected by the maintainer ("my own product, not a fork"). Option 2 dilutes the sovereign single-binary property (C1) and creates an external Node dependency (C2).

## Constraints (evaluation criteria)

- **C1 — Sovereign / single-binary Go**, on-prem, inference over NaN/Ollama, **never Claude in the fleet** (ADR-002 + Anthropic ToS).
- **C2 — Own synthesis, no fork**: reimplement patterns; depend on no one's base.
- **C3 — Dogfood-first**: first validation is building imagesensortool with iris.
- **C4 — The gate is a component, not the moat**: the differentiator is uniting the three layers.
- **C5 — Anti-stall**: build bottom-up, each step shippable, real forcing function (imagesensortool).
- **C6 — Licences are not a constraint** (reimplementation in Go is licence-agnostic).

## Build order (reframed, not restarted)

ADR-007's bottom-up order survives; the layers map onto it:

1. **Layer 1 — Runtime** (motor + NATS + pi worker + runtime adapter). *Task 1+2 done; Task 3-5 in progress (SDD-034a).*
2. **Layer 2 — Process/Gate** (typed verdict + bounded loops + worktree + anti-cheat). Goal: **first verified PR on imagesensortool**.
3. **Layer 3 — Meta-orchestration** (goal-ancestry + heartbeats + budgets + org-chart). Goal: from "one verified PR" to **building imagesensortool toward a goal**. Absorbs the resident-supervisor (#72).

## Pending per-layer deep-analysis (before implementing each layer)

The strategic decision above does not need these; implementing each layer does.

- **Layer 3 — Paperclip deep code analysis** (only an overview exists; deep-dive launched 2026-06-14). Blocks layer-3 design, not this ADR.
- **Layer 2 — swe-af deep code dive** (design analysis done; Apache code is liftable). Before implementing the gate.

## Consequences

**Positive:**
- One coherent product thesis: the factory, not a coding-loop feature. The moat (the synthesis) cannot be cloned by adopting any single donor.
- Task 1+2 (the Go motor) are the correct foundation under every layer — no rework.
- "Take the best of everyone" becomes the explicit, defensible strategy, not an accident.
- Dogfood-first validates bottom-up: a real product (imagesensortool / ImagingSuite) proves each layer.

**Negative / constraints:**
- Building all three layers in Go is more work than integrating Paperclip; layer 3 (meta-orchestration) is substantial net-new work.
- Polyglot persists (Go motor + pi/TS workers) — accepted since ADR-002.
- Reframes a 1-day-old ADR (007) and amends ADR-006's phasing; the roadmap must be re-sequenced.

## What would reopen this ADR

- Layer 3 deep-analysis reveals Paperclip's architecture is fundamentally incompatible with the "leader = black box" plug-in model → revisit the synthesis shape.
- A donor reference is superseded by a materially better one → re-audit (Regla del 3).
- The factory needs >50% human time on factory-suitable work (ADR-006 reopen trigger) → the factory thesis itself is re-examined.

## References

- Reframes: [adr-007-iris-v0-coding-loop](./adr-007-iris-v0-coding-loop.md) · Amends: [adr-006-pivot-ai-native-engineering-studio](./adr-006-pivot-ai-native-engineering-studio.md) · Builds on: [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md), [adr-003-license-apache-2](./adr-003-license-apache-2.md)
- References audited: agent-crew-analysis, swe-af-analysis (vault `25-prestudy/competition/`), Paperclip analysis (deep-dive in progress)
- Absorbs: mlorentedev/knowledge#72 (SDD-041 resident supervisor) → layer 3
- Paperclip: https://paperclip.ing · https://github.com/paperclipai/paperclip
