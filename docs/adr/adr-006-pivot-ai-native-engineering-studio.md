---
id: adr-006-pivot-ai-native-engineering-studio
type: adr
status: accepted
owner: manu
created: "2026-06-10"
tags: [strategy, positioning, studio, factory, imagingsuite, iris, pivot]
---

# ADR-006 — Pivot: AI-Native Engineering Studio (supersedes ADR-004 positioning)

> **Status:** Accepted, 2026-06-10
> **Supersedes:** [adr-004-positioning](./adr-004-positioning.md) (positioning / ICP / GTM only). Architecture siblings stay valid re-scoped to the internal machine: [adr-001-frontend-htmx-go](./adr-001-frontend-htmx-go.md), [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md), [adr-003-license-apache-2](./adr-003-license-apache-2.md), [adr-005-architectural-boundary](./adr-005-architectural-boundary.md).
> **Session record:** 2026-06-10-pivot-session-research-and-decision (4-stream web research + decision trail).

## Decision — DECIDED (irreversible minimum, signed 2026-06-10)

1. **Identity.** The business is an **AI-native engineering studio**: it sells **verified engineering outcomes** — starting vertical: **imaging / image sensors** — produced by an in-house **agent factory** over sovereign infrastructure, with Manu's expert sign-off as the variance control. Sellable software **crystallizes from delivery** (ImagingSuite first; later the machine itself). Not an "AI agency", not a token reseller, not an agent-framework vendor.
2. **Revenue ladder (sequence, not menu).**
   - **Phase 0 — ramp, now:** sell nothing. The factory proves itself: storefront (Wave 0) + EMVA 1288 core (Wave 1) + own-channel content. Invoice €0, risk ~0.
   - **Phase 1 — first revenue:** outcomes (EMVA characterization reports, spec-matching, application engineering; €k/deliverable or €3-8k/mo retainers).
   - **Phase 2 — post-jump:** sovereign stack deployments on customer infra (promotes candidate-vertical-sovereign-eu-regulated; €50-200k engagements). Gate: factory proven (Waves 0-1 shipped).
   - **Phase 3 — product:** ImagingSuite SaaS and/or the packaged machine ("AI-OS"). Gate: N≥2 deployments.
3. **Two-layer brand.** `mlorente.dev` = personal **authority engine** (vertical-agnostic, documents the factory, never invoices). The **studio** (name TBD ≤4 weeks) = the invoicing vehicle. The person endorses the studio; verticals are slots, not identity. Second vertical is **earned by winning the first**, never opened preemptively.
4. **iris re-scoped.** From FDE platform product → **internal factory engine**. The Go motor is **deferred until real coordination friction appears** (evidence: scaffolding is commodity — mini-swe-agent ~100 LOC ≈ near-SOTA). The factory starts on the existing harness (Claude Code + vault + skills + hermes).
5. **ImagingSuite promoted** to flagship product with dual role: factory testbed AND real product. **Clean-room rule:** built only from the public EMVA 1288 standard; no Teledyne-derived material ever enters the repo.
6. **Layer-0 OUT.** No GPU capex, no token resale, no hosted-inference business. NaN = supplier (€70/mo fuel for the factory) + market observatory; commercial fallback (DeepInfra-class, ~$0.3/Mtok) for anything customer-facing. **Independent of Cristian/Helmcode** (cordial, no pact).
7. **Teledyne guardrail (hard).** Zero customers in the sensor/imaging sector while employed. Formal employment-contract review during the ramp (re-opens OQ-12 "assume clean + proceed" for *business* use — that decision covered dogfood only). Public artifacts sanitized.
8. **Distribution = own-channel broadcast** (YouTube + web documenting what the factory builds, with verification evidence). NOT open-source build-in-public community development.
9. **Runway.** 6-12 month ramp while employed. Jump gate = first signed engagement or equivalent validated revenue signal (define numerically at Phase-1 entry).

## Working hypotheses (revisable at gates — no re-litigation before a gate fires)

| Hypothesis | Gate to revisit |
|---|---|
| Wave order: Wave 0 (storefront + 2-3 packaged products, hard cap) → Wave 1 (EMVA core spike) | Wave 0 stalls >2 weeks |
| Phase-2 offering shaped per candidate-vertical-sovereign-eu-regulated | Factory proven (Waves 0-1 shipped) |
| Phase-3 packaging (ImagingSuite SaaS / AI-OS) | N≥2 sovereign deployments |
| Studio naming | Decide ≤4 weeks (absorbs OQ-16 scope) |
| Wave 0 product candidates: hive, ts-bridge, pollex or pdf-modifier-mcp | Re-pick at Wave 0 kickoff |

## Context — evidence base (research 2026-06-10)

- **YC thesis cluster:** Spring 2026 RFS "AI-Powered Agencies" (Epstein — "use the software yourself and sell the finished product at 100x the price"); Summer 2026 RFS "AI-Native Services" + "AI OS" + "Company Brain" (Alströmer / Hu); Warren playbook episode (2026-06-03). Services-as-software margin arbitrage; **variance is the existential constraint** → answered here by verification harness + expert sign-off. Canonical: https://www.ycombinator.com/rfs
- **Landscape:** scaffolding is commodity; production winners sell outcomes in existing workflows with owned runtimes and eval discipline (Devin $492M ARR; Factory.ai $1.5B; Blitzy $1.4B — all closed-model). Open gap: **provable agent labor** (post-Builder.ai) + **maintenance under SLA**; open-model economics fund verification-heavy harnesses.
- **Market:** Spanish SME sovereign flat-fee company-brain slot is empty (German comparable: CompanyGPT €14,990 + €399/mo flat; no Spanish equivalent found). Channels: Kit Digital (€12k vouchers incl. AI), ALIA (€150M integration fund). Inference prices collapse ~10x/yr (a16z LLMflation; Gartner −90% by 2030) → **value migrates up-stack** (Christensen): governance, integration, verification, outcomes.
- **Helmcode/NaN decoded:** one person (Cristian Córdova, SRE, Madrid). Capex-ladder playbook (consulting → GPUs at OVH → flat-rate inference → paid community → agent hosting; each layer monetizes the same hardware twice). Factory claims exceed public evidence (no CI/review-gates published; tourism demo self-labeled "experiment"; agent_crew ~41 stars, stalled since April). Complementary once layer-0 is OUT — he holds the prosumer/community wing; this studio holds the on-prem/enterprise wing.
- **Revealed preference (vault evidence):** Manu ships what he uses or what has a team/community around it (NaN hackathon, FAE autopilot, hermes, kubelab); stalls on solitary builds for abstract buyers (iris repo never created despite OQ-18 "start execution" 2026-05-21). The factory therefore works **with an audience** (own channel) and **dogfood-first** sequencing.

## Closes / moots

- **OQ-14** (GTM first customer) → replaced by the revenue ladder.
- **OQ-16** (iris naming) → mooted; studio naming is the live decision.
- **MQ-3** (horizontal vs vertical wedge) → **conscious reversal of ADR-004**: vertical-first (imaging), because the operator model requires a domain where the founder's signature carries professional-liability weight, and the founder's moat intersection is maximal there.
- **2026-05-27 brainstorm:** candidate-vertical-sovereign-eu-regulated promoted to the Phase-2 lane (gated). All other kills/defers unchanged.

## Post-decision signals (watch list — dated, no decision changes)

- **2026-06-12 — Rafael Casuso thread (X) + CalliopeBI.** Thesis published: every company needs **one centralized AI** with **one source of truth**, **open-source heart** (cost + strategic-dependency at massive internal usage; private/secure by design), **analytical AND operational** (decides and executes, not just integrates/analyzes), coordinating **internal agents + human employees alike**, with an **external AI layer** exposing services/API/agents/MCP. He is evolving **CalliopeBI** toward this and recruiting a **design-partner program** for company pilots.
  - **Validates:** the company-brain thesis (§ Context — YC RFS cluster) restated independently from a BI-first angle; single-SSOT + open-source-core + analytical→operational ladder map point-for-point onto this studio's stack thesis.
  - **Challenges:** the § Market evidence line "Spanish SME sovereign company-brain slot is empty (no Spanish equivalent found)" — CalliopeBI is a Spanish contender now forming. Differential remains: they are BI/analytics-first and (apparently) cloud-first; this studio is sovereign-infra-first selling **verified engineering outcomes**, imaging vertical. Slot contested ≠ slot closed, but the empty-slot assumption now has a clock on it.
  - No decision change. Watch item: CalliopeBI design-partner traction; re-read at the Phase-2 gate (candidate-vertical-sovereign-eu-regulated).
- **2026-06-12 — iris backlog migrated to the bitácora board (#74-77); start being prepared.** The vault `11-tasks.md` (SDD-034a-i stream) was retired per VAULT-001/ADR-018 and its untracked items (SDD-034a/g/h/i) promoted to GitHub issues, in anticipation of starting iris. The §4 Go-motor deferral ("until real coordination friction appears") still stands as written; this flags that the start is being prepared. **Revisit §4** when the start is concrete — repo creation is the act that un-defers. No decision change yet.

## What would reopen this ADR

1. Jump gate not reached by 2027-06 → full strategy review.
2. Teledyne contract review returns adverse → re-sequence verticals (start non-imaging; ladder and identity survive).
3. Waves 0-1 show the factory needs >50% human time on factory-suitable work → the factory thesis itself is re-examined.

## Consequences

- 10_projects/iris/00-context re-scoped (internal engine); 10_projects/imagingsuite/00-context activated (flagship, clean-room); ADR-004 gets a superseded banner.
- Execution tracking → bitácora issues (Wave 0 storefront, Wave 1 EMVA spike, contract review) per ADR-018 — task state lives on the board, not in the vault.
- `nan-video-pipeline`: finish the hackathon team commitment this week; no new scope after.
- Frozen explicitly: iris Go motor (gate per §4), kubelab (substrate/maintenance), hive (maintenance; gets packaged in Wave 0), FAE autopilot (day-job tooling).
