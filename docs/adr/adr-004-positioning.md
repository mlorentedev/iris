---
id: adr-004-positioning
type: adr
status: superseded
created: "2026-05-21"
owner: manu
tags: [iris, positioning, strategy, fde, bowling-pin, moat, prestudy]
---

# ADR-004 — iris Positioning (v0 canonical)

> ⚠️ **SUPERSEDED 2026-06-10 by [adr-006-pivot-ai-native-engineering-studio](./adr-006-pivot-ai-native-engineering-studio.md).** The positioning, ICP, and GTM in this ADR are no longer canonical. Architecture siblings (ADR-002, ADR-005) remain valid as internal-machine specs. Kept for audit trail.

> **Status:** Accepted, 2026-05-21
> **Closes:** OQ-15 (cuándo crear ADR positioning), MQ-3 (v0 primary angle), partial OQ-13/OQ-14 (pricing + GTM frame)
> **Depends on:** [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md) (technical architecture), [adr-003-license-apache-2](./adr-003-license-apache-2.md) (license enables FDE customer-owned customizations)
> **Related:** helmcode-nan (locked thesis 2026-05-14), 12-market-position-product-architecture, 13-marketing-gurus-analysis, 14-dev-platform-teams-fde-model, 11-open-questions-tracker

## Decision

`iris` is a **Forward-Deployed-Engineer-grade AI agent platform** for **EU mid-market platform engineering teams (Series A-C) running K8s on-prem under DORA/NIS2**, sold as **bespoke engagement (€40-120k setup + €2-8k/mo recurring)** delivered **8-12 weeks embedded**, designed around a **knowledge SSOT versioned in the customer's own git** (the *vault-as-SSOT principle*, formalized in [adr-005-architectural-boundary](./adr-005-architectural-boundary.md)).

The v0 wedge is **horizontal dev/platform team productivity** (NOT business-vertical bespoke). Business-vertical packs (RMA, banca, calibration QA) emerge **organically post-customer-request** as v0.5+ expansion, NOT pre-architected.

## Scope

What this ADR covers:
- iris's market position statement and ICP definition
- Business model archetype (FDE-grade, NOT SaaS multi-tenant)
- v0 primary angle (horizontal dev/platform team) and v0.5+ expansion path (vertical packs organic)
- Bowling pin #1 (sharpened post-MQ-3 decision)
- MOAT inventory (M1-M6) and what differentiates iris vs commercial/indie alternatives
- Pricing bracket and value-ladder frame
- What iris explicitly is NOT

What this ADR does NOT cover:
- Technical architecture (lives in [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md) + [adr-005-architectural-boundary](./adr-005-architectural-boundary.md))
- Naming final iris vs codename (OQ-16, ~2-4 weeks decision)
- GTM execution tactics per customer (OQ-14, post-v0.3 dogfood validated)
- Specific vertical pack contracts (emerge v0.5+ per customer)

## Context

This ADR consolidates positioning material accumulated across the 2026-05-19+ prestudy brainstorm sessions (docs 01-14) plus the thesis lock in helmcode-nan (*"agentes en K8s on-prem a $200/mes vs $5000 cloud — monto y entreno tu platform team"*).

Two prior strategic shifts converge here:
1. **Stream P pivot 2026-05-09** (positioning) — *"Platform Engineering for AI Workloads"* category
2. **Orchestrator pivot 2026-05-15** (architectural materialization of Stream P) — multi-agent platform reframe

helmcode-nan is the thesis lock; this ADR is its positioning crystallization.

## Positioning statement (canonical)

**iris is an FDE-grade AI agent platform applying Palantir economics to EU mid-market regulated platform engineering teams.**

Expanded:

- **What it is:** A knowledge-SSOT-centric AI agent platform where agents cultivate the customer's own git-versioned vault. Coordinator (Go, deterministic) orchestrates 4 v0 knowledge-domain agents (capture + apply + adr-auto-capture + post-incident-learner) running on Hermes substrate. Deployed on-prem K8s.

- **Whom it's for (ICP v0):** EU platform engineering teams at Series A-C startups running K8s on-prem with DORA/NIS2 compliance pressure, who need AI workflows their compliance posture can absorb and whose data sovereignty needs preclude cloud-managed agent platforms.

- **How it's delivered:** FDE engagement model (Palantir-style). Senior engineer (Manu, future hires) embedded 8-12 weeks. Stand up vault as SSOT + deploy iris on customer K8s + configure agents to feed vault + train platform team to curate. Customer owns vault + customizations forever (Apache 2.0).

- **What outcome it produces:** Platform team executes 3-5x more "AI workflow"-class tasks (ADR drafting, post-incident learning, knowledge surfacing) without hiring; vault becomes operational SSOT that compounds value over time; DORA/NIS2 audit evidence accrues as side-effect of normal operation.

## ICP v0 definition

| Dimension | Spec |
|---|---|
| **Geography** | EU (España + Portugal primary, France/Germany/Netherlands secondary) |
| **Stage** | Series A-C startups, ~50-300 engineering employees, ~€10-100M ARR |
| **Vertical** | Tech / regulated SaaS / industrial / banca tech / health tech — anyone under DORA/NIS2/GDPR pressure |
| **Tech stack** | K8s on-prem or hybrid (NOT pure-cloud-managed). Git as code SSOT. Already running platform team (NOT engineering manager wearing 3 hats) |
| **Buyer persona** | Platform team lead / engineering manager / VP Engineering (NOT CTO, NOT CIO procurement) |
| **Pain trigger** | "Our platform team is the bottleneck. We can't hire fast enough. Cloud-managed agent platforms (Bedrock, CF Managed Agents) don't fit our sovereignty/regulatory posture." |
| **Budget signal** | Can sign €40-120k engagement without board approval (= midmarket Series B+) |
| **Language** | ES/EN bilingual native preferred (Manu's compositional moat) |

## Bowling pin #1 (sharpened)

> **EU platform engineering teams (Series A-C) running K8s on-prem under DORA/NIS2 who need AI workflows with knowledge SSOT versioned in their own git repo, delivered FDE-style and customer-owned forever.**

Refinements vs original Moore-style proposal (doc 13):
- Added **K8s on-prem** constraint (matches Manu's 20yr moat literally)
- Added **knowledge SSOT in customer's own git** (vault-as-SSOT differentiation vs Glean SaaS / CF Managed)
- Added **FDE-style + customer-owned forever** (Apache 2.0 + bespoke delivery)
- Removed industrial-sensor-specific framing (was too narrow; horizontal v0 angle absorbs it)

## Business model — FDE-grade (NOT SaaS)

| Metric | Target v0 | Reference |
|---|---|---|
| **Engagement type** | Bespoke 8-12 weeks embedded delivery | Palantir / Anthropic FDE / Scale AI |
| **Setup pricing** | €40-120k per engagement | 14-dev-platform-teams-fde-model § 4 economics |
| **Recurring pricing** | €2-8k/mo (maintenance + extensions + retraining) | Industry standard FDE post-engagement support |
| **Annual customer count v0** | 3-5 customers | Founder-led delivery ceiling |
| **5-year scaling target** | 5-10 FDEs, 50-100 customers max | FDE hiring is the scaling constraint, not infra |
| **Margin model** | Premium (€500-1500/day FDE bill rate × engagement weeks + product premium) | NOT SaaS multiples |

**Investor narrative (when relevant):** *"We are an FDE-grade AI agents company applying Palantir economics to EU mid-market regulated. Bowling pin = platform engineering teams. Profitable founder-led shop, not a SaaS unicorn. Premium positioning + deep customer integration + sovereignty differentiation."*

## MOAT inventory (M1-M6)

| ID | Moat | Mechanism | Defensibility |
|---|---|---|---|
| **M1** | Knowledge accumulator | Vault grows compounding per customer; switching costs become enormous after 6-12 months of curated knowledge | High — knowledge is non-portable in practice |
| **M2** | EU + K8s + AI compositional | Manu's 20yr K8s + 15% AI + bilingual ES/EN = rare combo. Hard to replicate without Manu-equivalent senior hire | High — talent scarcity in EU |
| **M3** | On-prem K8s sovereignty | Cloud-managed competitors (Bedrock/Vertex/CF Claude Managed) cannot serve customers with on-prem mandate | Structural — vendors will not pivot off-cloud |
| **M4** | Vault-as-SSOT customer ownership | Customer's knowledge lives in their own git repo. Apache 2.0 + Owned-by-customer = zero lock-in. Paradoxically increases stickiness because customer trusts the engagement | High — counter to SaaS lock-in playbook |
| **M5** | FDE-grade delivery + knowledge transfer | Premium engagement model; customer's team trained = customer becomes the operational owner. Replicable but slow scale = barrier to competition | Medium — slow-scaling FDE shops are not VC-attractive, reduces attacker pressure |
| **M6** | DDD-para-IA fine-tuning (per vertical pack, v0.5+) | Fine-tuned LoRA per customer's bounded context (Unsloth/Axolotl pipeline). Bounded model per business — differentiator vs generic-model competitors (Glean/Hebbia/Mistral/Cohere) | High when active — customer's fine-tuned model is non-portable |

## What iris explicitly is NOT

- **NOT a SaaS multi-tenant platform.** Single-tenant per-customer deployment. No shared infrastructure.
- **NOT a chat assistant or generic AI agent runtime.** iris is the orchestration + knowledge layer ABOVE substrate (Hermes + OpenCode + Ollama). Substrate is adopted; iris is the integration + delivery + customer-bespoke layer.
- **NOT a cloud-managed service.** Customer K8s on-prem mandatory v0. Cloud-K8s acceptable v0.5+ if customer regulatory posture permits.
- **NOT a generic platform engineering automation tool** (e.g., Backstage, Crossplane). iris is AI-agent-native; platform-eng automation tools handle different problem space.
- **NOT a workflow builder** (n8n, Temporal, Airflow). iris coordinator is deterministic Go + YAML routing; workflow builders use DAG-builders. n8n is *integration target via MCP*, NOT substrate (TQ-10 decided).
- **NOT a personal AI assistant** (OpenClaw, cherry-studio Layer 1). Different ICP entirely.
- **NOT vertical CX automation** (Cresta, Decagon, Sierra). Different problem space.

## Why this positioning (vs alternatives considered)

| Alternative considered | Why rejected |
|---|---|
| Vertical bespoke first (RMA/banca/calibration QA priority) | Manu's compositional moat is **maximum in horizontal platform engineering**, indirect in business verticals. Dogfood signal (Manu's kubelab) is daily for platform work, weekly/monthly for business processes. MQ-3 decided híbrida C: horizontal v0 + vertical organic v0.5+. |
| SaaS multi-tenant + freemium | Sovereignty requirements of ICP preclude shared infra. SaaS multiples not achievable with FDE constraint. Glean/Hebbia/Mistral already occupy this space. |
| Open-core (free core + paid enterprise features) | Apache 2.0 + FDE delivery is cleaner. Open-core requires gating features (compliance, multi-tenant, scale) that customer needs in v0 anyway. Open-core also signals "you need premium for serious use" — anti-aligned with FDE premium positioning. |
| Generic agent platform (compete with Bedrock / CF Managed) | Cannot win cloud-managed on cloud-managed terms. Counter-positioning required: on-prem + sovereignty + customer-owned + FDE delivery. |
| Vertical-only with no horizontal angle | Loses Manu's dogfood validation loop. Risk: spending 8-12 weeks on customer #1 before any personal-use signal. |

## Consequences (what this changes downstream)

| Area | Change |
|---|---|
| **[adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md)** | Re-validated as accepted. Architecture (Go motor + Python workers + NATS + HTMX) is consistent with FDE delivery + on-prem K8s + customer-owned. No amendment needed; status reaffirmed. |
| **[adr-005-architectural-boundary](./adr-005-architectural-boundary.md)** | Direct sibling — formalizes the *vault-as-SSOT principle* + CORE/EDGE boundary that this positioning depends on. |
| **Backlog SDD-034 stream** | v0 implementation scope confirmed: 4 knowledge-domain agents (Combo 8) targeting Manu's kubelab dogfood as primary validation. SDD-034d agents = capture + apply + adr-auto-capture + post-incident-learner. |
| **Pricing pages / collateral (future)** | Drop SaaS-tier framing entirely. Lead with FDE engagement + €40-120k brackets + customer-owned-forever messaging. |
| **GTM v0.3+** | Kubelab content (Manu's existing audience) + NaN network + peer engineering communities EU. ICP filter: K8s on-prem + Series A-C + EU + regulatory pressure. Disqualify pure-cloud + non-EU + non-regulated. |
| **Hiring plan** | 5-year horizon: 5-10 FDE-profile senior engineers (NOT SaaS engineers). Profile: 10+ yr K8s + AI-curious + bilingual + customer-facing senior. |
| **OQ-13 pricing concrete** | Closed at €40-120k setup + €2-8k/mo recurring bracket. Per-engagement quote based on customer scope; published bracket is anchor. |
| **OQ-14 GTM first customer** | Frame deferred to post-v0.3 (after kubelab dogfood validates 4 agents). v0.3 trigger: KGR + HITL accept + "would I miss it" thresholds per [adr-005-architectural-boundary](./adr-005-architectural-boundary.md) § metrics. |
| **OQ-16 naming final** | Still open. *iris* is codename; final name decision in ~2-4 weeks per orchestrator-terminology. Positioning statement is naming-agnostic (works with any final name). |

## Cross-references

- helmcode-nan — thesis lock 2026-05-14 (origin of positioning)
- 12-market-position-product-architecture — market position raw analysis
- 13-marketing-gurus-analysis — Dunford/Hormozi/Moore/Brunson stress-test
- 14-dev-platform-teams-fde-model — FDE economics + dev/platform team agents
- 11-open-questions-tracker — open questions tracker (MQ-3 decided here)
- [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md) — technical architecture (sibling)
- [adr-005-architectural-boundary](./adr-005-architectural-boundary.md) — architectural boundary contract (sibling, formalizes substrate adopt vs build)
- orchestrator-terminology — naming convention (iris = codename, final TBD)
- 2026-05-15-orchestrator-pivot — Stream P+ pivot 2026-05-15
- 2026-05-27-brainstorm-6-business-ideas — 6-idea evaluation 2026-05-27; canonical ICP re-validated, "iris-as-IAP" captured as alt GTM framing (not amending this ADR), Sovereign EU vertical accepted as candidate (candidate-vertical-sovereign-eu-regulated)
