---
id: adr-005-architectural-boundary
type: adr
status: accepted
created: "2026-05-21"
owner: manu
tags: [iris, architecture, boundary, vault-ssot, core-edge, substrate, prestudy]
---

# ADR-005 — iris Architectural Boundary Contract (CORE/EDGE + vault-as-SSOT)

> **Status:** Accepted, 2026-05-21
> **Closes:** DP-1, DP-2, DP-3, DP-4, DP-5, TQ-1, TQ-3 (via TQ-15), TQ-8, TQ-12, TQ-13, TQ-14, TQ-15, OQ-9, OQ-10, OQ-11, EQ-1 (all from 11-open-questions-tracker)
> **Depends on:** [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md) (technical foundation). Business positioning lives in the vault strategy reports, not the repo.
> **Related:** 10-feature-extraction-catalog (13 architectural axes + seed decisions), pattern-dual-memory (evolved into 3-tier knowledge flow), lesson_agentmemory_validation (their pivot = our axiom), orchestrator-terminology

## Decision

iris is governed by the **vault-as-SSOT principle**: the customer's git-versioned vault is the canonical store for every artifact with long-term value (knowledge, agent contracts, audit snapshots, training corpora). Postgres and other operational stores are **regenerable indices**, not sources-of-truth. claude-mem is an upstream observation buffer.

iris's boundary between **CORE (built)** and **EDGE (adopted substrate)** is contractual, not aspirational: iris CORE is the coordinator + agent contract + HITL primitive + KB schema + audit chain. iris EDGE is Hermes + vault MCP + claude-mem + pgvector + NATS + mcp-go + acpx + bernstein primitives + LiteLLM + unstructured-io + Unsloth/Axolotl + msgraph (v0.5+).

v0 ships **4 knowledge-domain agents** (Combo 8: capture + apply + adr-auto-capture + post-incident-learner), validated by **Knowledge Growth Rate + HITL accept rate + qualitative "would I miss it" rating**.

## Scope

What this ADR covers:
- The vault-as-SSOT principle (axiomatic; everything below derives from it)
- iris CORE (what iris builds) vs iris EDGE (what iris adopts) boundary
- The three-tier knowledge flow (claude-mem → iris agents → vault)
- v0 agent inventory + contract surface (Combo 8 quartet)
- Coordinator design (DP-1: deterministic Go + LLM advisor edge cases)
- Agent contract YAML format (DP-2: declarative + escape hatch)
- KB layer pattern (DP-3: vault directories as 3-layer)
- Memory tiering (DP-5: 2-tier Postgres+pgvector OR SQLite-vec)
- Identity/audit phasing (DP-4: hooks v0, cert packaging v0.5, dossier v1)
- Protocol surface v0 (MCP + ACP + NATS internal; A2A deferred — EQ-1)
- v0 validation metrics + exit criteria (OQ-11)

What this ADR does NOT cover:
- Component-level architecture (lives in [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md))
- Frontend specifics (lives in [adr-001-frontend-htmx-go](./adr-001-frontend-htmx-go.md))
- Positioning + business model (lives in the vault strategy reports, not the repo)
- Implementation task breakdown (lives on the bitácora board, SDD-034 stream)
- Naming final iris (OQ-16, ~2-4 weeks decision)
- Per-customer vertical pack specs (emerge v0.5+ organically)

## Context

This ADR consolidates 5 discussion points (DP-1..5), 3 substrate questions (TQ-1, TQ-8, TQ-9), and 4 pre-ADR open questions (OQ-9, OQ-10, OQ-11, EQ-1) decided across sessions 2026-05-19 → 2026-05-21 captured in 11-open-questions-tracker.

The catalyst was the **vault-as-SSOT realization 2026-05-21**: every prior architectural decision had a missing meta-principle. Articulating *"vault is SSOT, all else is regenerable index"* aligned all 5 DP decisions, resolved 3 substrate ambiguities, and made the validation metric (OQ-11) mechanically derivable from `git log`. This ADR is the principle's formal home.

The business positioning this architecture serves lives in the vault (strategy reports); this ADR carries only the architecture.

---

## Section 1 — The vault-as-SSOT principle

**Axiom:** *The customer's git-versioned vault is the canonical store for every artifact with long-term value. Postgres (and any other operational store) is a regenerable index. If you lose Postgres tomorrow, iris reconstructs it in <10 minutes from vault. If you lose vault, you lose iris's identity.*

### What lives in vault (SSOT)

| Artifact class | Vault path pattern |
|---|---|
| Knowledge artifacts | `<project>/lessons/`, `<project>/patterns/`, `<project>/playbooks/`, `<project>/decisions/adr-*.md` |
| Agent contracts | `<project>/agents/<agent-name>.yaml` |
| Customer vertical packs (v0.5+) | `<customer>/{config,kb,agents,playbooks}/` |
| Audit chain snapshots (DP-4) | `<project>/audit/<date>.signed.md` (periodic emission from Postgres) |
| Fine-tuning training corpora (EQ-2 v0.5+) | derived projection: `git archive HEAD <project>/` filtered |
| Project-meta + governance | `_meta/`, `00_meta/patterns/`, frontmatter via `types.json` (schema) |

### What does NOT live in vault (regenerable)

| Artifact | Where it lives | Why NOT vault |
|---|---|---|
| In-flight workflow state | Postgres / Redis | Mutable per-second, no historical value |
| HMAC bytes audit chain | Postgres append-only | Binary-dense; snapshot summary to vault daily |
| pgvector embeddings | Postgres pgvector | Derived from vault content; regenerate in minutes |
| Hermes session context window | Hermes runtime memory | Ephemeral execution-layer state |
| Telemetry hot-path raw events | Loki / Postgres | Rollup weekly to vault `<project>/telemetry/<week>.md` |
| Secrets (customer API keys) | SOPS / HashiCorp Vault | Encrypted; vault holds references, not values |
| Inter-agent NATS messages | NATS in-memory + JetStream | Ephemeral inter-process; envelope-by-envelope durability not required |

### The rule of thumb

*"If you lose it and it hurts → vault. If you lose it and recompile in <10 min → not vault."*

### Why this principle (vs alternatives)

| Alternative considered | Why rejected |
|---|---|
| Postgres as SSOT (iris-native schema) | Vendor-coupling on iris's own DB; customer cannot easily inspect/own/export. Defeats the customer-owned-forever principle. |
| External DB-as-SSOT (Snowflake/BigQuery/etc.) | Cloud-coupling. Violates ICP sovereignty requirement. |
| swarmvault-managed KB store | Adoption-cost (need iris's wrapper) + competes with iris's identity (DP-3 decided Inspirado, not adopt-as-engine). Vault is simpler + more sovereign. |
| Hybrid (some artifacts vault, some Postgres) | Boundary becomes fuzzy → "where is decision X stored?" requires checking 2 systems. Single SSOT discipline simplifies operations + customer mental model. |

### Vault-write concurrency policy (R3 closure, 2026-05-22)

If the four Combo 8 agents fire on overlapping triggers (cron + slash + webhook + file-watch), they may attempt to write to the same vault file in the same second — most likely targets are `90-lessons.md`, `<project>/decisions/_index.md`, and per-project lessons indices. The OQ-11 exit criterion *"0 vault corruption / concurrency bugs in 8-week window"* is only mensurable if a serialization policy exists. This subsection is that policy.

**Decision:** all agent-originated vault writes go through a **single serialized writer** owned by the iris motor. Agents NEVER touch the vault filesystem directly. The writer:

1. **Receives write intents over a dedicated NATS subject** `team.<team>.vault-write` (one consumer per deployment — a `WorkQueue` JetStream consumer with `max_ack_pending = 1`). This subject is reserved in candidate-nats-protocol-envelope subject taxonomy and MUST NOT be used by any other producer.
2. **Serializes writes per repository.** The writer's worker pool size is `1` for the customer's vault repo. Multiple distinct customer vaults (v0.5+) get one writer worker each — never one worker for two repos.
3. **Wraps each write in an advisory `flock(2)` on `<vault>/.iris-write.lock`** for defense-in-depth against a misconfigured deployment that spawns a second writer (operator error, container restart race). `flock` is process-level, sufficient for the single-container motor design (ADR-002 § 1).
4. **Performs the operation as `git add <files> && git commit -m "iris: <agent> <slug>"`** — atomic relative to the working tree. If pre-commit hooks reject (gitleaks, vault-validate.py), the writer surfaces the error to HITL gate, NEVER force-commits or skips hooks.
5. **Records each write in the audit chain** (§ 9) keyed by `(agent_id, vault_path, git_sha_pre, git_sha_post, hmac)` — this is the durability primitive that lets OQ-11 measure incidents via `git log` + audit log diff.

**What this rules out:**
- A worker writing to a checked-out vault on the worker's filesystem (workers are stateless per ADR-002 Component 2).
- Two motor instances pointing at the same vault (forbidden by deployment topology; one iris per customer, per the R6 contractual clause).
- File-level CRDT or git-merge resolution — both add complexity for a problem that single-writer serialization solves trivially in v0.

**Failure mode:** if the writer process dies mid-commit, the next motor restart sees the JetStream WorkQueue with unacked messages; the writer replays them. Idempotency is guaranteed because each draft carries a `MessageID` (envelope) and the writer dedupes by checking `git log --grep="iris-draft-<MessageID>"` before applying. Worst case: a draft commit lands twice with identical content — visible in git log but harmless (HITL gate filters dupes at review time).

**Trigger to re-evaluate:** if vault corruption metric (OQ-11) shows >0 incidents in any 8-week window, or if multi-vault customers (v0.5+) exceed 5 deployments per motor instance, revisit with subject-per-vault sharding or external write queue (NATS KV).

---

## Section 2 — CORE/EDGE boundary

### iris CORE (built)

| Component | Built because | Reference |
|---|---|---|
| **Coordinator (Go, deterministic + YAML routing)** | Identity-defining: DP-1 zero-LLM-in-coordination-loop + DORA/NIS2 certifiable. No commercial off-the-shelf matches this profile. | [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md) Component 1 |
| **Agent contract YAML schema + parser** | Identity-defining: DP-2 declarative + 4-stage cognitive loop + HITL gates + audit refs. Custom schema enforced by iris validator. | This ADR § 4 |
| **HITL gate primitive** | Identity-defining: deliberative-not-reactive principle. Reusable across agents. Implementation: vault PR-or-status workflow. | This ADR § 4 |
| **KB schema (3-layer raw/derived/schema via vault dirs)** | Identity-defining: DP-3 Inspirado-not-Adopted (informed by swarmvault but iris-native). Customer-readable structure. | This ADR § 7 |
| **Audit chain (HMAC-SHA256 + ed25519 agent cards)** | Identity-defining: bernstein-inspired but iris-implemented (controlled by iris, not external lib runtime risk). | This ADR § 9 |
| **HTMX console + operator API** | Operator-facing surface; cannot delegate (defines iris's UX identity). | [adr-001-frontend-htmx-go](./adr-001-frontend-htmx-go.md) |
| **MCP server (iris-as-tool exposure)** | Customer integration surface; iris must own contract to maintain compat. | This ADR § 10 |
| **Trigger plane (cron + webhook + file-watch + claude-mem-session-end)** | Operational logic specific to iris; not commodity. | [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md) Component 1 |

### iris EDGE (adopted substrate)

| Substrate | Capa | Rol v0 | Rationale TQ-9 |
|---|---|---|---|
| **Hermes** (Nous Research, 160k⭐, MIT) | execution | LLM runtime per-agent (4 instances v0) | TQ-1 decided adopt-as-substrate execution-only; iris coord orquesta múltiples Hermes |
| **OpenCode** | execution | Coding-agent runtime per Manu's stack | orchestrator-terminology |
| **Ollama** | execution | Local LLM serving when applicable | orchestrator-terminology |
| **vault MCP (Hive)** | KB I/O | Single I/O surface to vault for all 4 agents | TQ-9; already in Manu's stack |
| **claude-mem** | session-memory upstream | Observation buffer for capture + adr-auto-capture agents (read-only) | TQ-2; coexist per pattern-dual-memory |
| **PostgreSQL + pgvector (or SQLite-vec)** | infra (ephemeral cache) | Workflow state + embeddings + audit-chain bytes (snapshot to vault) | DP-5 2-tier confirmed |
| **NATS JetStream** | infra | Inter-process messaging coord ↔ agents | [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md) Component 3 |
| **mark3labs/mcp-go** | protocol | MCP server impl in Go (iris MCP server exposed) | TQ-9 |
| **openclaw/acpx** | protocol | ACP CLI client for editor/coding-agent integration | TQ-9 |
| **bernstein primitives (HMAC + Ed25519 JWS)** | crypto | Audit chain crypto via library, NOT runtime adoption | TQ-9 (maintainer-risk medium, attention flag) |
| **LiteLLM** | LLM provider abstraction | Proxy multi-provider unified API | TQ-9 |
| **unstructured-io** | file ingestion | PDF/DOCX/PPTX/audio ingest for KB layer | TQ-9 |
| **Unsloth + Axolotl + vLLM + HF transformers/PEFT** | fine-tuning (v0 personal, v0.5+ customer packs) | EQ-2 DDD-para-IA pipeline | TQ-9 + EQ-2 |
| **msgraph-sdk-go + Entra ID** | M365 connector (v0.5+) | SharePoint/OneDrive/Teams/Outlook/Calendar via Graph API | TQ-11 scheduled v0.5+ |
| **kubernetes-sigs/agent-sandbox** | K8s runtime substrate (Apache 2.0, K8s SIG Apps, 2.3k⭐, v1beta1) | K8s pod lifecycle controller — each iris agent = 1 `Sandbox` CR managed by SIG controller; iris coord client-go wraps CRD | OQ-1 spike 2026-05-21 — podTemplate embeds full corev1.PodSpec; hermes-agent example native in repo; saves ~60% SDD-034c K8s adapter effort |
| **n8n** | integration target (NOT substrate) | Customer's n8n calls iris via MCP | TQ-10; iris exposes MCP server, n8n is one consumer |

### Out-of-scope for iris (different problem space)

Per TQ-8 clarification: NEVER frame these as "iris alternatives":
- Claude Code, Codex, Cursor — Manu's personal dev tools, OUT of platform scope per orchestrator-terminology
- OpenClaw, cherry-studio (Layer 1 personal AI assistants) — different ICP
- Cresta/Decagon/Sierra (vertical CX) — different problem space

### Canibalizadores (compete in same identity slot — DO NOT adopt as substrate)

Per TQ-8: products that occupy the slot iris wants to own. Patterns may be studied; code/substrate is not adopted.

- swarmclaw — coordinator/runtime layer (overlaps iris Hermes-orchestration role)
- swarmvault — KB-spine layer (overlaps iris Eje 1; DP-3 decided Inspirado-not-Adopt)
- bernstein — compliance/audit layer (overlaps iris Eje 12; primitives adopted as library only, not runtime)
- mission-control — dashboard/governance layer (overlaps iris Eje 10; 32 panels patterns adopted, code is not)
- kiwiq — multi-tier memory + orchestration (overlaps Ejes 4+5; pattern lessons adopted via DP-5)
- Letta — memory specialist (overlaps Eje 5)
- Bedrock/Vertex Agents, CF Claude Managed — managed orchestrators (cloud-locked; iris non-cloud)
- Full enterprise knowledge-platform suites — different commercial bracket and problem space

### Criterion definitivo (per Manu correction 2026-05-20)

> *"Se pueden adoptar productos ya existentes sin tener que hacer todo desde cero. No tiene sentido hacer un LiteLLM si ya existe."*

The boundary is **NOT library-vs-product**. It is **compete-with-identity vs serves-as-substrate**. iris's identity = *knowledge accumulator + agentes como interfaz + customer delivery + vertical bespoke + EU bilingual*. Any product that does NOT compete with this identity and resolves a sub-problem is adoptable — even if it is a product, not a library.

### Substrate adoption guarantees + fallback triggers (R2 closure, 2026-05-22)

Adopting an external substrate concentrates external dependency risk. The OQ-1 spike verdict ADOPT for `kubernetes-sigs/agent-sandbox` is sound but rests on a `v1beta1` CRD whose upstream stability is not guaranteed. The K8s SIG history (`meta.k8s.io/v1beta1`, `policy/v1beta1`) shows that v1beta1 APIs can be redesigned or removed entirely on the path to v1. This subsection establishes the contract under which iris adopts agent-sandbox and the explicit triggers that force re-evaluation.

**Adoption guarantee required from any K8s-substrate adoption:**

1. **Contract pin.** iris pins the agent-sandbox CRD version it depends on in `internal/runtime/k8s/agent_sandbox_version.go` (constant + Go-embed of the CRD schema for offline validation). Bumping the pin requires explicit ADR amendment.
2. **CI contract test before v0.3.** SDD-034c MUST include an integration test that deploys the pinned `Sandbox` CRD into a kind/k3d cluster and exercises the iris client-go wrapper (create → status → terminate). The test runs on every PR touching `internal/runtime/k8s/`. Without this test, the adoption is theoretical and the next K8s SIG release can break iris invisibly.
3. **Parallel Docker adapter stays first-class.** ADR-002 Component 4 lists Docker + Kubernetes + local-process. The Docker adapter is NOT a stepping stone for the K8s adapter — it is the primary delivery substrate for v0 consultoría engagements. The K8s adapter is opt-in. If agent-sandbox vanishes, customers on Docker substrate are unaffected.

**Re-evaluation triggers (any one fires the ADR amendment process):**

- **T1 — Upstream API change.** agent-sandbox CRD ships a breaking change between minor releases (e.g., field rename, removal, semantic shift in `podTemplate`). iris pins to the last-good release and opens an ADR amendment scoped to: stay pinned vs. migrate vs. fork-with-attribution vs. fall back to iris-native K8s adapter.
- **T2 — Maintenance signal degradation.** No commits to `kubernetes-sigs/agent-sandbox` `main` in the prior 90 days OR a single maintainer covers >80% of merged PRs over a 90-day window. Either is a yellow flag triggering a 30-day watch period; two consecutive 90-day windows is an amber flag forcing the amendment process.
- **T3 — Stalled progression to v1.** If agent-sandbox remains at `v1beta1` 18 months after iris first adopts it (so by ≈ 2027-11-22), open an amendment to either (a) adopt the iris-native K8s adapter from the candidate doc as the canonical path and demote agent-sandbox to optional, or (b) confirm continued reliance with explicit risk acceptance.
- **T4 — Compliance objection.** A customer compliance review rejects agent-sandbox due to non-GA status. iris falls back to the iris-native K8s adapter for that customer + opens amendment to evaluate broader migration.

**Fallback path (always available):** the iris-native K8s adapter design lives in candidate-agent-runtime-capabilities. SDD-034c MUST keep that path implementable within ≤2 weeks of effort even after agent-sandbox adoption — meaning no iris-internal interface change is allowed that would only make sense for the agent-sandbox shape. The `AgentRuntime` interface stays substrate-neutral.

### Pattern adoption ledger (2026-05-25)

Competitive intel scan 2026-05-25 surfaced four products in adjacent identity space (gbrain, OB1, Cognee, Letta) and one substrate candidate (Graphiti). Audit details live in 11-open-questions-tracker TQ-12 through TQ-15. This subsection records pattern-level adoption decisions that update iris's CORE feature surface.

**Adopted patterns (no code lift; iris-native re-implementation under Apache 2.0):**

| Pattern | Source | Maps to | Effort estimate |
|---|---|---|---|
| Synthesis-with-citations output format | gbrain | § 4 Eje 1 + agent contract output mandate | ~1-2 days |
| Gap-analysis primitive ("brain doesn't know X yet") | gbrain | § 7 KB layer output decorator | ~1 day |
| Self-wiring KG typed edges (`deployed`, `triggered`, `caused_incident`, `triages`, `references`, `supersedes`) | gbrain | § 7 KB schema | ~3-5 days |
| Dream cycle daemon (sync → extract_facts → consolidate → synthesize) | gbrain | § 5 coord scheduled jobs (SDD-034h) | ~5-7 days |
| Protected operations boundary (trusted local vs remote MCP) | gbrain | § 9 audit chain + § 1 vault-write policy | ~1-2 days |
| BrainBench-style evals (separate `iris-evals` sub-repo) | gbrain | testing infra cross-cutting | ~3-5 days |
| `llms.txt` + `llms-full.txt` + AGENTS.md install protocol | gbrain | dev tools integration | ~1 day |
| `cycle_locks` table concurrency primitive | gbrain | § 1 OG-4 vault-write reinforcement | ~1 day |
| Primitives/Extensions/Recipes/Skills/Integrations/Schemas/Dashboards taxonomy | OB1 | vertical-pack file layout (converges TQ-4) | converges existing decision |
| `thoughts` extensibility model ("schemas extend not replace") | OB1 | § 7 KB schema vertical-pack extensions | ~1-2 days |
| Schema-Aware Routing pattern | OB1 | EDGE Eje 2 connectors | ~2-3 days |
| Wiki Compiler (graph-backed entity pages, on-demand regen) | OB1 | § 7 KB derived output + console rendering | ~2-3 days |
| Aiception agent-level skill creation + HITL gate (closes TQ-3) | OB1 | § 4 Combo 8 agent capability extension | ~3-5 days |
| Typed reasoning edges + Opus/Haiku tier classifier | OB1 | § 7 KB edges + DP-1 LLM advisor tier routing | ~2-3 days |
| Agent Memory API contract reference | OB1 | § 8 Eje 5 contract surface | ~1 day (read-and-formalize) |
| World Model Diagnostic (20-min readiness, FDE pre-engagement) | OB1 | business deliverable, NOT architecture | post-v0.3 template |
| Multi-AI assistant entry points (Skill + GPT + GEM) | OB1 | GTM tactical channel post-v0.3 | post-v0.3 GTM |
| Memory-blocks naming (`core` / `recall` / `archival`) | Letta | § 8 Eje 5 schema labels | trivial |
| Ontology grounding pattern | Cognee | § 7 KB vertical-pack schema extensions | ~1-2 days |

**Rejected substrate / pattern candidates (recorded for traceability):**

| Candidate | Reason rejected |
|---|---|
| Graphiti (getzep) as KB-spine engine | (1) Neo4j/FalkorDB/Kuzu/Neptune dep — violates DP-5 Postgres-only zero-deps; (2) mandatory LLM extraction violates DP-1 zero-LLM hot path; (3) `posthog>=3.0.0` core dep violates EU sovereignty default; (4) Zep-CLA contributor agreement license risk. DP-3 Inspirado-not-Adopted reaffirmed |
| mem0 | Overlap with Eje 5 2-tier (DP-5) without incremental value |
| Letta runtime adoption | Contradicts DP-1 (agent self-edit memory = LLM-in-coord). Only naming pattern adopted |
| Cognee GraphRAG retrieval as engine | iris uses hybrid recursive-CTE + pgvector instead |
| OB1 code lift | NOASSERTION license blocks derivative code use. Patterns extracted only |

**Operational discipline:** any future substrate adoption must pass the same audit shape — verify stack consistency (DP-5 Postgres-only) + DP-1 zero-LLM-coord alignment + EU sovereignty + license cleanliness BEFORE adoption is recorded. See feedback_polish_vs_build_trap — adoption decisions are capped, not open-ended.

---

## Section 3 — Three-tier knowledge flow

This supersedes the 2-tier pattern-dual-memory formalization. With iris in the picture, the canonical flow is:

```
┌────────────────────────────────────────────────────────────────┐
│ Tier 1 — RAW OBSERVATION BUFFER (upstream sources)            │
│                                                                 │
│   claude-mem (auto-captured Claude Code session observations)  │
│   git log (commits, PRs, code changes)                         │
│   Hive vault baseline (existing crystallized knowledge)        │
│   Customer connectors (SharePoint v0.5+, custom v0.5+)         │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
                          ▼ read-only
┌────────────────────────────────────────────────────────────────┐
│ Tier 2 — iris CURATION LAYER (4 agents + HITL gates)           │
│                                                                 │
│   capture  → drafts lesson/pattern from delta                  │
│   apply    → surfaces KB to current task (passive)             │
│   adr-auto-capture  → drafts ADR from PR/chat decisions        │
│   post-incident-learner → drafts playbook from post-mortems    │
│                                                                 │
│   Coordinator (Go deterministic) + Hermes per-agent execution  │
│   HITL gate before any vault write (Manu/operator approves)    │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
                          ▼ write (HITL-approved)
┌────────────────────────────────────────────────────────────────┐
│ Tier 3 — VAULT SSOT (crystallized, git-versioned, customer-   │
│                       owned, durable forever)                  │
│                                                                 │
│   Knowledge: lessons, patterns, ADRs, playbooks, decisions     │
│   Agent contracts: <project>/agents/*.yaml                     │
│   Audit snapshots: <project>/audit/<date>.signed.md            │
│   Vertical packs (v0.5+): <customer>/{config,kb,agents}/       │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
                          ▼ derived (regenerable)
              pgvector embeddings, fine-tuning corpora,
              Postgres workflow indices
```

---

## Section 4 — v0 agent inventory + contract surface (OQ-9 + OQ-10)

v0 ships **Combo 8 = 4 knowledge-domain agents**. All 4 share the same contract surface; differ in trigger + Hermes profile + KB-write target.

| Dimension | **capture** | **apply** | **adr-auto-capture** | **post-incident-learner** |
|---|---|---|---|---|
| **Trigger** | cron 24h + `/iris-capture` slash | file-watch Claude Code session active + `/iris-apply` | git PR webhook + claude-mem session-end signal | file-create on `<project>/post-mortems/` + `/iris-learn-incident` |
| **Input** | claude-mem obs 24h + git delta + vault delta | task signal (file path, prompt, recent git activity) | PR diff + PR discussion / chat transcript | post-mortem markdown + linked metrics/logs |
| **Output artifact** | drafted lesson/pattern entry (`status: draft`) | ranked KB snippets inline (MCP response) | drafted ADR (template, numbered) | drafted playbook entry (structured) |
| **KB-read scope** | full vault + claude-mem corpus + git history | full vault + claude-mem + ADR index + embeddings | existing ADRs + related code/docs | existing playbooks + related incidents (semantic) + runbooks |
| **KB-write scope** | `90-lessons.md`, `<project>/lessons|patterns/` | telemetry-only (snippets surfaced/clicked, HMAC-signed) | `<project>/decisions/adr-NNN-<slug>.md` | `<project>/runbooks/` or `_meta/patterns/incident-<slug>.md` |
| **HITL gate** | 1 — pre-merge entry draft | 0 (passive surfacing) | 1 — pre-merge ADR draft | 1 — pre-merge playbook draft |
| **Hermes profile** | summarization-tuned (LLM distill; v0.5 fine-tune candidate via EQ-2) | retriever + reranker (small LLM + embedding) | decision-classifier + template-filler (structured output) | narrative→structured + pattern extractor |
| **Substrate deps** | claude-mem reader + git CLI + vault MCP + Hermes | iris MCP server + pgvector + vault MCP + Hermes | git/GitHub webhook listener + claude-mem reader + vault MCP + Hermes | file watcher + vault MCP + Hermes |
| **Audit chain hooks (DP-1+DP-4)** | HMAC-SHA256 on draft + ed25519 agent card sig | HMAC log of retrieval events (which snippets shown) | HMAC on ADR draft + ed25519 sig | HMAC on playbook draft + ed25519 sig |
| **Failure mode** | claude-mem empty → skip + log; LLM timeout → retry 1x, defer 24h | KB miss → return empty silently; LLM down → keyword search fallback | no decision signal → skip silently; classifier confidence <0.7 → no draft | post-mortem <500 chars → skip; ambiguous → flag manual |

### Why these 4 (vs other combos considered)

Combo 8 maximizes 4 simultaneous criteria:
1. **Daily-dogfood signal** — 3 of 4 agents fire on Manu's daily kubelab activity
2. **Thesis validation end-to-end** — KB-spine + agents-as-interface + coord deterministic + HITL all exercised
3. **Hermes substrate validation** — all 4 agents are LLM-heavy (no deterministic-only agents to dilute Hermes validation)
4. **v0.5+ foundation** — adr-auto-capture seeds Eje 12 audit chain + decision lineage; post-incident-learner seeds ops vertical pack; capture+apply reusable across every v0.5 vertical

Combos rejected:
- 3 agents knowledge-only — no operational angle; MQ-3 horizontal demo weak
- 5 agents incl. gitops-drift/helm-maintainer/runbook-executor — cluster credentials + security surface complicate v0; violates polish-vs-build trap
- 4 agents incl. codebase-navigator/pre-pr-reviewer — requires code-RAG infra (parser+chunker+embeddings); defer v0.5
- 3 agents incl. compliance-audit-collector — too narrow; compliance evidence emerges latently from adr + post-incident without dedicated agent

### Output format mandate (synthesis-with-citations + gap-analysis, 2026-05-25)

Per TQ-13 (gbrain pattern adoption), every Combo 8 agent's `output_type` MUST satisfy two format requirements v0:

1. **Synthesis-with-citations** — the agent's output is synthesized prose (or structured artifact derived from prose) where every factual claim carries a citation reference to the source page/file/snippet that supports it. Format: `[ref: <path>#<anchor>]` inline or in a structured citations block. iris is NOT "return 10 chunks for the operator to read" — iris is "return the answer with sources verifiable"
2. **Gap-analysis primitive** — every output ends with an explicit `## Heads-up` section that states what the brain does NOT know yet about the topic. Example: *"No vault entries about X since 2026-04-22 (6 weeks). Source Y was not ingested in this run. Consider asking the operator before assuming Z is current."* This converts iris's blind spots into surfaced asks rather than silent omissions

**Provider constraint:** the LLM that performs synthesis MUST be invoked via LiteLLM proxy (TQ-9 substrate) using `hermes | opencode | ollama` providers. NEVER claude per [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md) § "Why no claude provider". gbrain's Anthropic-default does NOT carry over.

**Aiception agent-level skill creation (TQ-15 / closes TQ-3):** any of the Combo 8 agents MAY invoke an explicit `skill_create` capability that proposes a new skill (YAML + prompt + tool bindings) when its `adapt` stage detects a repeated pattern in failure modes. The proposed skill MUST traverse a HITL gate before activation; activation writes the new skill to `<project>/agents/skills/` and emits an audit chain entry. The coord NEVER autonomously creates skills (DP-1 zero-LLM-in-coord preserved).

---

## Section 5 — Coordinator design (DP-1)

**Decision:** Hybrid deterministic — Go coordinator with YAML routing rules + bernstein-inspired HMAC audit chain + LLM advisor only in `_default` fallback (output LOGGED, NOT authoritative; final decision recorded as iris coord deterministic action).

### Why hybrid (vs pure-zero-LLM or pure-LLM-supervisor)

| Pattern | Tradeoff |
|---|---|
| Zero-LLM (pure bernstein) | Deterministic + audit-trivial + DORA/NIS2 certifiable + Go implementable; rigid for unanticipated cases |
| LLM supervisor (AWS Bedrock) | Flexible + adapts to novelty; un-auditable + costly + latency variable |
| **Hybrid (chosen)** | Deterministic for matched patterns (80-90% v0 cases) + LLM advice for `_default` fallback (logged but not authoritative) → maintains certifiability while allowing graceful unknown-handling |

### Coordinator contract

- YAML routing rules versioned per agent: pattern → action mapping
- HMAC-SHA256 chain on every routing decision (audit chain, DP-4 hooks v0)
- LLM advisor invoked ONLY when routing rule = `_default`; its suggestion is recorded in audit chain with classification `advisor_suggestion`, but the coordinator's final action is the deterministic fallback path (e.g., HITL escalation or skip)
- Routes cover 80-90% v0 horizontal dev/platform team cases (matches MQ-3 alignment)

---

## Section 6 — Agent contract YAML (DP-2)

**Decision:** Declarative YAML format with 4 stages (`understand` / `plan` / `execute` / `adapt`) + HITL gates + audit chain references + escape hatch via `script_runner` stage when declarative is insufficient.

### Why declarative + escape hatch

| Rationale |
|---|
| DP-1 deterministic coordinator routes by pattern matching → requires declarative agent contracts |
| Manu's K8s 20yr background = K8s IS declarative; aligns naturally |
| Multiple production platforms validate the declarative pattern |
| Escape hatch (`script_runner` stage) preserves flexibility for edge cases without sacrificing the 90%+ declarative path |

### Format sketch (full schema lives in SDD-034d implementation)

```yaml
# <project>/agents/capture.yaml
id: capture
type: agent
version: v0
exposed_protocols: [mcp, acp]   # EQ-1: A2A deferred v0.5+

trigger:
  cron: "0 0 * * *"    # 24h
  slash: /iris-capture

stages:
  understand:
    inputs:
      - claude-mem://observations?since=24h
      - git://log?since=24h
      - vault://changed?since=24h
    
  plan:
    llm: hermes
    profile: summarization-tuned
    
  execute:
    output_type: draft_entry
    template: lesson_or_pattern
    
  adapt:
    on_failure: retry_then_defer_24h
    
hitl_gates:
  - id: pre_merge_draft
    type: vault_status_workflow
    
audit:
  hmac_chain: required
  agent_card_sig: ed25519
```

---

## Section 7 — KB layer 3-layer pattern (DP-3)

**Decision:** swarmvault-inspired (NOT swarmvault-adopted) 3-layer structure implemented via vault directory conventions + Obsidian frontmatter as schema enforcement.

### Layer mapping

| swarmvault concept | iris implementation |
|---|---|
| `raw` | `<project>/raw/` — claude-mem export rollups, git log captures, post-mortem unprocessed |
| `derived` | `<project>/lessons/`, `<project>/patterns/`, `<project>/playbooks/`, `<project>/decisions/` — agent drafts post-HITL |
| `schema` | Obsidian frontmatter + `types.json` (vault-wide schema enforcement; already exists) |

### Why iris-implemented (not swarmvault-adopted)

- Adopting swarmvault as KB engine = iris's identity becomes swarmvault-wrapper, weakening positioning (M1 MOAT defensibility)
- Customer's vault is Obsidian-native (standard markdown + frontmatter) — no proprietary engine required to read it
- Sovereignty maximized: customer can leave iris and keep their vault fully usable
- Lessons from swarmvault studied + applied as design influence (the 5-reference distillation: swarmvault + memoir + kiwiq + bernstein + Letta)

---

## Section 8 — Memory tiering (DP-5)

**Decision:** 2-tier — Postgres state + pgvector embedded (or SQLite-vec for single-binary install). Redis cache v0.5 if latency emerges. Separate vector DB v1+ if scale justifies.

### Tier breakdown

| Tier | Store | Contents | Refresh |
|---|---|---|---|
| **Hot (ephemeral cache)** | Postgres + pgvector embedded | Workflow state, embeddings, agent registry parsed from vault, audit-chain bytes (snapshot to vault daily) | Real-time; regenerable from vault in <10 min |
| **Cold (SSOT)** | vault (git) | All durable artifacts per § 1 | Append-only via HITL gate; full git history |

### Why 2-tier (vs kiwiq's 4-tier or Letta-style memory blocks)

- Opción C horizontal v0 ICP = Manu's kubelab dogfood = laptop/single-K8s. Zero-deps is core UX.
- kiwiq's 4-tier (Postgres + MongoDB + Weaviate + Redis) is over-engineering for v0; emerges if/when customer scale demands it
- mission-control F25.4 zero-deps pattern aligns with iris's positioning

---

## Section 9 — Identity/audit phasing (DP-4)

**Decision:** Split across v0/v0.5/v1 phases. v0 includes audit-chain HOOKS (HMAC-SHA256 + ed25519 agent cards) but NOT certification packaging. v0.5 adds per-artifact customer-key signing + DORA/NIS2 evidence packaging templates. v1 adds automated compliance dossier generation per customer audit.

### Phasing detail

| Phase | What's included | Trigger |
|---|---|---|
| **v0** | HMAC-SHA256 audit chain on every coord decision + ed25519 agent card signing per agent.yaml + HMAC sigs on agent outputs | Personal kubelab dogfood (no certification customer yet) |
| **v0.5** | Per-artifact customer-key signing + DORA/NIS2 evidence packaging templates + signed audit snapshots emitted to vault | First customer engagement requesting certification posture |
| **v1+** | Automated compliance dossier generation per customer audit cycle | Recurring customer audit cycles materialize |

### Why phased (vs all-v0 or all-defer)

- All-v0 (full certification packaging): 2-3 months effort that no v0 user needs; violates polish-vs-build trap
- All-defer (no hooks v0): retrofit cost when first cert customer arrives is high; cheaper to wire hooks now
- Phased: v0 hooks are cheap (~3-5 days of implementation), preserve future productizability mandate, no wasted v0 effort

### Key management policy (R10 closure, 2026-05-22)

The audit-chain pretensión *"DORA/NIS2 certifiable"* (§ 9 v0.5+) depends on a key management policy that, until now, was implicit. A certifiable audit chain is only as strong as the worst link in its key lifecycle. This subsection makes the policy explicit at v0; certification packaging (v0.5) extends it.

**Key inventory v0:**

| Key | Role | Storage | Rotation cadence |
|---|---|---|---|
| **HMAC-SHA256 chain key** (`audit_chain_hmac`) | Seals every coord decision in the audit chain; used to verify chain integrity on read | SOPS-encrypted under a customer KMS key (Age recipient by default); referenced from vault as `<project>/secrets/audit_chain_hmac.sops` | **90 days** by default; customer may shorten via contract |
| **Ed25519 agent card signing key** (`agent_card_<id>_priv`) | Signs each `<project>/agents/<id>.yaml` agent card to bind agent identity | SOPS-encrypted, one keypair per agent contract; public key embedded in agent.yaml frontmatter | **365 days** by default; rotated on agent.yaml content change |
| **Per-artifact customer signing key** (v0.5+) | Signs released artifacts (audit snapshots, training corpora) with a customer-owned key | Customer-supplied KMS / HSM reference | Customer-defined |

**Rotation procedure (v0, automatable):**

1. iris emits a `system_command` envelope of type `audit.rotate_hmac` 7 days before the scheduled cadence (rotation pre-warning visible to operator in HTMX console).
2. On rotation day, the iris motor generates a fresh key, writes it SOPS-encrypted to the new versioned path (`audit_chain_hmac.v<N>.sops`), and updates the `current` pointer atomically.
3. The audit chain emits a `rotation` envelope sealed with the **old** key, containing the SHA-256 of the **new** key's public counterpart (a self-attesting handoff). This makes the rotation itself a verifiable event on the chain.
4. The old key is retained for the verification window (default 18 months) — the chain must remain verifiable across rotation boundaries.
5. After the verification window expires, the old key is **zeroized** via SOPS `delete` + an audit-chain entry recording the zeroization.

**Compromise procedure (manual, declared incident):**

1. Operator marks key compromised via HTMX console `Rotate now (compromise)` action.
2. Motor immediately rotates per the procedure above, but the old key is **NOT** retained — it is zeroized at rotation, and the audit chain marks all entries sealed under the compromised key with `provenance: degraded`.
3. iris emits an incident envelope to `<project>/audit/<date>-incident-key-compromise.signed.md` (signed by the NEW key) recording the compromise window + the SHA-256 of any logged entries during that window.
4. Required follow-up: the post-incident-learner agent (§ 4) is auto-triggered to draft a playbook entry covering the response.

**What is NOT done v0:**

- HSM / TPM-backed keys (deferred to v0.5+ when customer compliance requires it).
- Multi-party rotation (e.g., 2-of-3 operator approval). Single operator key authority v0; multi-party deferred to v0.5+ certification packaging.
- Cross-deployment key rotation orchestration. Each iris deployment rotates independently.

**What this rules out at the design level:**

- A single HMAC key whose lifetime equals the deployment lifetime. Compliance audits routinely flag this; v0 prevents the failure mode by mandating rotation cadence even before a certifying customer exists.
- Logging keys to Loki or any EDGE-adopted telemetry. § 1 already excludes secrets from regenerable stores; this subsection reaffirms that audit keys are vault references, not vault values, and never leave SOPS-encrypted form except in motor memory at use time.

**Trigger to expand v0.5+:** first customer engagement requesting DORA/NIS2 certification posture. Adds: HSM-backed primary key, customer-owned KEK wrapping iris HMAC keys, per-artifact customer signing, dual-control approval for rotation outside cadence.

---

## Section 10 — Protocol surface v0 (EQ-1 + EQ-4)

**Decision:** v0 ships MCP + ACP + NATS internal. A2A deferred to v0.5+ trigger (first customer explicitly requests Vertex/Bedrock/CF Managed Agents interop).

### Protocol responsibilities

| Protocol | Resolves | iris role v0 |
|---|---|---|
| **MCP** (Anthropic) | Tools/connectors exposed to/from agents | iris ships MCP server (mcp-go); 4 agents speak MCP for tool/KB access |
| **ACP** (OpenClaw + emerging) | Editor/coding-agent interop | iris speaks ACP via acpx for Claude Code session integration (apply agent uses) |
| **NATS internal** (ADR-002) | iris coord ↔ agents | proprietary internal; not exposed externally |
| **A2A** (Google, 2025) — DEFERRED | Cross-platform agent-to-agent | NOT implemented v0; YAML hook (`exposed_protocols`) preserves option |

### Hook for v0.5+ A2A

agent.yaml format includes `exposed_protocols: [mcp, acp]`. Extending to `[mcp, acp, a2a]` is non-breaking; v0.5 implementation is ~1-2 weeks of adapter work (A2A is JSON-RPC over HTTP, not deep architecture).

---

## Section 11 — Validation metrics + v0 exit criteria (OQ-11)

### Metrics

| Metric | Definition | Computation | Threshold v0 |
|---|---|---|---|
| **Knowledge Growth Rate (KGR)** — primary | Net-new vault entries accepted from iris-source per week | `git log --since="1w" --diff-filter=A --name-only --grep "source: iris-" \| wc -l` | ≥5 net entries/week sustained 4 weeks |
| **HITL Accept Rate** — secondary per agent | drafts_accepted / drafts_created, rolling 4 weeks | Frontmatter-tracked on each draft + acceptance commit | ≥70% across all 4 agents |
| **"Would I miss it?" rating** — qualitative weekly | Manu 1-5 per agent: *"if this agent disappeared tomorrow, would I miss it?"* | Manual weekly | ≥3/5 per agent at week 8 |
| **Vault corruption / concurrency bugs** | Merge conflicts, dirty state, write races | Git status + iris audit log | 0 incidents in 8-week window |

### v0 exit criteria

If all 4 metrics hit threshold for required window → v0 validated → gate opens to first FDE customer engagement.

If 1+ fails → iterate the failing agent(s) (DP-2 declarative YAML enables fast iteration without rebuilding Go motor) before customer engagement.

### Metrics explicitly rejected

| Metric | Why rejected v0 |
|---|---|
| Task completion time lift | Requires A/B baseline; N=1 dogfood = noise > signal. Apply v0.5+ |
| Apply hit rate (snippets clicked) | Telemetry-heavy; qualitative "would I miss it" covers v0 |
| Cost per agent run (€/$) | Premature optimization; measure when signal-of-fit confirmed |
| Vault search-replacement rate | Telemetry-heavy; defer |
| NPS / customer satisfaction | N=1 dogfood; applies v0.5+ |
| MAU / DAU agent usage | Vanity metric — accepted outputs > raw runs |

---

## Consequences (what this changes downstream)

| Area | Change |
|---|---|
| **[adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md)** | Reaffirmed accepted. Component-level architecture unchanged. This ADR-005 layers the strategic boundary on top: which components are CORE (built) vs which compose external substrate. |
| **SDD-034 stream tasks** | SDD-034d (agents) scope locked to 4-agent Combo 8 quartet. agent contract YAML schema specified per § 4-6. Implementation iteration enabled via DP-2 declarative. SDD-034c K8s runtime adapter scope shrinks ~60% post OQ-1 = agent-sandbox adopted as K8s substrate (each agent → 1 `Sandbox` CR; iris wraps client-go). Docker + local-process adapters stay iris-built. |
| **SDD-034a Phase 3 bootstrap** | Now write-ready with substrate adopt list (§ 2 EDGE table); features.json can be populated per CORE/EDGE classification. |
| **pattern-dual-memory** | Superseded by 3-tier knowledge flow (§ 3). Update pattern to reference this ADR as authoritative. |
| **10-feature-extraction-catalog Eje 1, 4, 5, 7, 10, 12** | All seed decisions formalized here. Document can be archived to historical reference once vault audit pass. |
| **11-open-questions-tracker** | DECIDED via this ADR: DP-1..5, OQ-1 (substrate spike 2026-05-21), OQ-9, OQ-10, OQ-11, OQ-12 (Teledyne IP 2026-05-22), EQ-1, EQ-4, TQ-1, TQ-3 (via TQ-15 / Aiception 2026-05-25), TQ-8, TQ-12, TQ-13, TQ-14, TQ-15. Remaining open (8 buckets): EQ-5, OQ-2, OQ-3, OQ-16, TQ-4, TQ-5, TQ-6, TQ-11, MK-1..9. |
| **OQ-1 spike (substrate K8s SIG audit)** | Next research action post this ADR. Audit `kubernetes-sigs/agent-sandbox` for v0 substrate compatibility; if it fits, swap iris-built Docker runtime adapter for it (saves ~6-8h impl); if not, iris-native runtime per adr-002 stays. |
| **OQ-10 SDD-034d task spec** | Per § 4 contract surface, SDD-034d = "implement coord routing + 4 agent.yaml + HITL gate primitive + audit chain hooks + 4 trigger types". Effort estimate ~2-3 weeks post SDD-034a Phase 3 done. |
| **Validation gate before customer engagement** | Per § 11 exit criteria. First customer FDE engagement requires v0 validation pass (8-week dogfood window minimum). |

## Alternatives considered (architectural-level)

| Alternative | Why rejected |
|---|---|
| **iris-as-KB-engine (no vault dependency)** | Defeats customer-sovereignty positioning. Customer would need iris running to read their own knowledge. |
| **Single all-purpose agent (vs 4 specialized)** | Loses validation of multi-agent coordinator contract surface. DP-1 deterministic coord becomes theoretical. |
| **LLM-supervisor coordinator (LLM in the coordinator)** | Loses DORA/NIS2 certifiability + the audit-trivial property. |
| **Imperative agent definitions (code-first)** | Coord cannot route deterministically. DP-1 + DP-2 must align; chose declarative for both. |
| **kiwiq 4-tier memory (Postgres + Mongo + Weaviate + Redis)** | Over-engineered v0; zero-deps positioning compromised. Expand v0.5+ if needed. |
| **No audit-chain v0 (full defer)** | Retrofit cost when first cert customer hits is too high. Hooks now, packaging later. |
| **MCP only v0 (no ACP)** | Loses Claude Code integration day-1 (apply agent's primary signal). ACP is cheap (acpx adopted). |
| **3 protocols v0 (incl. A2A)** | Cognitive cost violates simplicity. ICP doesn't need v0. Hook preserves option. |

## Cross-references

- [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md) — technical architecture foundation (components, NATS, runtime adapters)
- [adr-003-license-apache-2](./adr-003-license-apache-2.md) — license enabling customer-owned customizations
- [adr-001-frontend-htmx-go](./adr-001-frontend-htmx-go.md) — frontend (operator console)
- 10-feature-extraction-catalog — 13 architectural axes + seed decisions (formalized here)
- 11-open-questions-tracker — open questions tracker (decisions consolidated here)
- pattern-dual-memory — predecessor pattern (superseded by 3-tier flow § 3)
- lesson_agentmemory_validation — agentmemory pivot validates vault-as-SSOT axiom
- orchestrator-terminology — naming conventions
