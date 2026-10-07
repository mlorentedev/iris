---
id: adr-012-ephemeral-worker-lifecycle-meeseeks-pattern
type: adr
status: proposed
created: "2026-09-22"
owner: manu
tags: [iris, workers, lifecycle, worktrees, meeseeks, naming, ephemeral]
extends: [adr-002-orchestrator-architecture, adr-007-iris-v0-coding-loop, adr-009-worker-runtime-go]
related: [adr-011-goal-lifecycle-crd-vs-postgres]
---

# ADR-012 — Ephemeral Fleet Worker Lifecycle & Semantic Naming Protocol (The Meeseeks Pattern)

> **Status:** Proposed, 2026-09-22
> **Extends:** [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md), [adr-007-iris-v0-coding-loop](./adr-007-iris-v0-coding-loop.md), [adr-009-worker-runtime-go](./adr-009-worker-runtime-go.md)
> **Prior art:** `lukehedger/meeseeks` (Personal Software Factory pattern: tasks → isolated worktrees → ephemeral agents → PR lifecycle)

## Context

In `iris`, ADR-007 and ADR-009 established that fleet workers run headless `pi` instances inside isolated git worktrees to resolve labelled issues. However, two operational details remained unspecified:
1. **Worker Lifecycle Dynamics:** How workers manage single-task lifecycles, handle PR review feedback, and guarantee clean environment teardown without leaving orphaned state or dangling worktrees.
2. **Fleet Observability & Identification:** How humans and supervisory systems identify and distinguish concurrent running workers in terminal multiplexers, logs, NATS subjects, and Slack notifications without relying on opaque UUIDs.

Prior art from Luke Hedger's `meeseeks` demonstrates the effectiveness of the "Mr. Meeseeks" paradigm for software development: agents exist solely to accomplish a single atomic task, run in dedicated worktrees, react to PR reviews, and self-terminate upon task completion.

## Decision

We establish the **Meeseeks Pattern** as the canonical execution and lifecycle model for all `iris` fleet workers:

### 1. Ephemeral Worker Lifecycle

Every fleet worker invocation follows a strict 5-phase lifecycle (phase 4 only runs when review asks for changes):

1. **Spawn (Birth):** Upon claiming a Goal/Issue from NATS (or local CLI trigger), the motor provisions an isolated git worktree at `../wt-<worker-id>` on branch `iris/<worker-id>`. The worker receives a semantic codename and boots clean.
2. **Context Ingestion:** The worker does not retain long-term state. Instead, it queries the `hive-vault` MCP server (`vault_query`) for project context (~50 lines) and relevant patterns.
3. **Execution & Gate (Mission):** The worker drives `pi` to implement the change, runs local tests, generates a typed self-check, and submits a PR. The two-stage verification gate (QA agent + human sign-off per ADR-007) evaluates the work.
4. **Respawn on Review (Survival Condition):** If QA issues a `FIX` verdict or a human leaves change requests on GitHub, the worker is respawned/re-engaged with the review comments as context.
5. **Teardown (Poof / Demise):** Once the PR merges (or is rejected/closed), the worktree is automatically unlinked and moved to trash/pruned, any running containers exit cleanly, and any operational discoveries are committed to the knowledge base via `hive`'s `capture_lesson`.

```
Issue Triggered
      │
      ▼
┌──────────────┐      queries context      ┌──────────────────┐
│  Spawn (wt)  │ ────────────────────────> │  hive-vault MCP  │
└──────┬───────┘                           └──────────────────┘
       │
       ▼
┌──────────────┐
│  Execute pi  │ ──> Open PR + Self-Check
└──────┬───────┘
       │
       ▼
┌──────────────┐   FIX / Review comments   ┌──────────────────┐
│  Gate Review │ ────────────────────────> │ Respawn Meeseeks │
└──────┬───────┘                           └────────┬─────────┘
       │ APPROVE + Merge                            │
       ▼                                            │
┌──────────────┐   capture_lesson                   │
│ Poof / Prune │ <──────────────────────────────────┘
└──────────────┘
```

### 2. Semantic Naming Protocol (`meeseeks-<produce>-<hash>`)

Workers are dynamically assigned names following the format `meeseeks-<produce>-<short-id>`. The produce surname indicates the operational profile, model routing, and urgency:

| Category | Surnames (Examples) | Routing / Target | Use Case |
|---|---|---|---|
| **Hot / Urgent** | `jalapeno`, `habanero`, `wasabi` | Frontier model (Claude Sonnet/Opus) | Critical bug fixes, broken CI, blocking triage |
| **Light / Edge** | `raspberry`, `blueberry`, `lime` | Local / Small model (Qwen 7B via Ollama) | Minor refactors, typo fixes, docs, chore bumps |
| **Crisp / QA** | `cucumber`, `celery`, `radish` | Structured / Reasoning model | Test suite expansion, verification, linter passes |
| **Dense / Rich** | `avocado`, `pumpkin`, `squash` | High-context model | Complex multi-file feature scaffolding |

### 3. Fast-Track CLI Execution (Interactive Mode)

In addition to the full NATS + K8s daemon pipeline, `iris` provides a lightweight local CLI subcommand (`iris up <prompt>` or `iris meeseeks <issue-id>`):
* Automatically provisions the local worktree.
* Spawns the worker in a terminal tab/pane.
* Polls PR reviews via `gh api` and cleans up on completion.

## Consequences

- **Positive:** Eliminates state drift; each task runs in a pristine sandbox.
- **Positive:** Clear, human-friendly identification across Slack, terminal multiplexers, and git branches.
- **Positive:** Lowers barrier to dogfooding: developers can use the Meeseeks loop locally without deploying the full K8s/NATS cluster.
- **Governance:** Workers must strictly clean up their worktrees upon exit (integrated with worktree sweepers).
