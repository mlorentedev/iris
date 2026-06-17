---
id: adr-003-license-apache-2
type: adr
status: accepted
created: "2026-05-16"
owner: manu
tags: [iris, orchestrator, license, apache, legal]
---

# ADR-003 — iris License: Apache 2.0

> **Status:** Accepted, 2026-05-16
> **Closes:** SDD-032
> **Context:** D18 scope clarification — `iris` is for freelance + corporate consultoría delivery, NOT SaaS multi-tenant open-core
> **Terminology note:** This ADR concerns `iris` (codename for our orchestrator platform — see orchestrator-terminology). It does NOT concern Hermes (Nous Research's autonomous agent runtime, MIT-licensed by upstream), which is one of the runtimes `iris` manages.
> **Related:** [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md), [adr-001-frontend-htmx-go](./adr-001-frontend-htmx-go.md), 2026-05-15-orchestrator-pivot, agent-crew-analysis, orchestrator-terminology

## Decision

`iris` will be released under **Apache License 2.0**. A `LICENSE` file containing the full Apache 2.0 text will be present at the repo root from the first commit. All source files will carry the SPDX header:

```
// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Manu Lorente / kubelab
```

## Scope

What this ADR covers:
- iris's Go motor source code
- iris's Go fleet worker source code (ADR-009)
- iris's frontend source code (HTMX + Go templates per [adr-001-frontend-htmx-go](./adr-001-frontend-htmx-go.md))
- Templates, configurations, deployment manifests shipped as part of iris

What this ADR does NOT cover:
- **Hermes (Nous Research)** — already MIT, upstream decides
- **OpenCode** — separate project, separate license
- **Claude Code CLI** — Manu's personal professional dev tool (used to develop iris), NOT a runtime iris manages, NOT bundled
- **Ollama** — separate project, separate license

## Context

The license question surfaced during the orchestrator pivot (D18 scope clarification — see orchestrator-terminology): iris is for freelance + corporate consultoría delivery, not SaaS multi-tenant open-core. That scope decides the license.

What remains relevant about license: (1) iris must not incorporate genuinely third-party copyleft (AGPL/GPL/SSPL) source, which would cascade its obligations onto the whole iris codebase; (2) whether consultoría client deliverables can be kept proprietary or modified privately without redistribution obligations (yes, under Apache).

## Options considered

| License | Permits commercial use | Patent grant | Redistribution obligation | Suitable for consultoría deliverables | Verdict |
|---|---|---|---|---|---|
| **Apache 2.0** | Yes | **Yes (explicit)** | Attribution + NOTICE file only | Yes — client keeps modified copy without source disclosure | **CHOSEN** |
| MIT | Yes | No (implicit only) | Attribution only | Yes | Acceptable; rejected for missing patent clause |
| BSD-3 | Yes | No | Attribution + no-endorsement | Yes | Acceptable; effectively MIT-equivalent here |
| GPL-3.0 | Yes | Yes | **Full source on distribution** | NO — every client gets GPL obligation cascade | Rejected |
| AGPL-3.0 | Yes | Yes | **Full source on distribution AND network use** | NO — strongest copyleft | Rejected |
| BSL (Business Source) | Time-bombed (4 years to OSS) | — | Restricted commercial use initially | Overcomplicated for consultoría scope | Rejected |
| SSPL | Yes | — | **All accompanying SaaS code must be open** | Wrong tool — designed for AWS-style SaaS resale defence | Rejected |
| Proprietary (closed) | Internal-only | — | None | Yes | Rejected — kills "self-hosted no lock-in" wedge messaging |
| Dual-license (Apache core + commercial Pro) | Yes / per terms | Yes | Per tier | Premature — no Pro tier exists, no paying client validates demand yet | Deferred |

## Why Apache 2.0 over MIT

Both are permissive and both meet the consultoría requirements. Apache 2.0 was chosen for three concrete reasons:

1. **Explicit patent grant.** Contributors granting patent rights protect the project from a future scenario where a contributor (or their employer) asserts patent claims against users. MIT does not address patents; users rely on implicit license-by-conduct, which courts have not consistently upheld. For a system orchestrating AI agents — a domain with high patent-troll activity (model serving, MoE routing, agent coordination patents) — explicit grant is cheap insurance.

2. **Corporate procurement familiarity.** Apache 2.0 passes most enterprise OSS-policy review without escalation. MIT also does, but Apache's NOTICE/LICENSE file convention and the explicit terms read as more "professional" to procurement reviewers in the consultoría target market (mid-size companies with at least an informal OSS policy).

3. **Same defaults as Kubernetes, Terraform, Hadoop, Spark.** iris's positioning ("self-hosted, K8s-aligned, infra-grade") inherits credibility by matching the license of the infra it sits alongside.

## Why NOT AGPL

**What AGPL requires** (per §13): a modified version of AGPL-licensed software, when made available to users over a network, must offer those users access to the corresponding source code. The trigger is **modifying AGPL-licensed code AND offering it over a network**.

**The real reason iris is not AGPL:** if iris incorporated genuinely third-party AGPL source code, the entire iris codebase that touches it would have to be released under AGPL, and iris would also be subject to the §13 network-use trigger. This conflicts with (a) shipping client deliverables that the client can modify privately, and (b) the practical reality that consultoría clients sometimes have policies against AGPL components. Hence iris must not incorporate third-party AGPL/GPL/SSPL source — enforced by the CI license gate below.

## Why NOT dual-license (yet)

Dual-licensing (Apache for community + commercial license for Pro tier) is the textbook open-core path (Sentry, Grafana, MongoDB historically). It was deferred — not rejected — for now:

- No paying client has validated demand for a Pro tier.
- No feature has been identified that is materially valuable enough to gate behind commercial license.
- Adding a CLA + commercial license framework before product-market fit adds friction (contributors must sign CLA, contributions are dual-assigned) without revenue justifying the overhead.
- Re-licensing later from Apache → Apache+Commercial-Pro is straightforward at the moment a Pro feature lands. Re-licensing in the opposite direction is not. The reversible-default principle favors Apache today.

## Consequences

**Positive:**
- iris can be deployed by anyone for any use without legal review.
- Genuinely third-party copyleft (AGPL/GPL/SSPL) source **cannot** be copied into iris — enforced by the CI license gate.
- Consultoría clients receive deliverables they can modify and operate without source-disclosure obligation.
- Patent grant protects iris and its ecosystem.

**Negative / constraints:**
- No moat against a competitor forking iris and SaaS-reselling it under a different name. This is acceptable given current scope; will be revisited if SaaS resale becomes a real threat (likely never for consultoría delivery scope).
- Every source file needs the SPDX header at first commit. Add a pre-commit hook to enforce.
- NOTICE file must be maintained as third-party dependencies accumulate. Use `go-licenses` (Go) and `pip-licenses` (Python) in CI to catch drift.

## Implementation checklist

When the iris repo is initialized (rename from `ai-workloads-starter` to `kubelab/iris`):

1. Create `LICENSE` at repo root with full Apache 2.0 text from [https://www.apache.org/licenses/LICENSE-2.0.txt](https://www.apache.org/licenses/LICENSE-2.0.txt).
2. Create `NOTICE` at repo root: `iris. Copyright (c) 2026 Manu Lorente / kubelab. Licensed under the Apache License, Version 2.0.`
3. Add SPDX header to every source file: `// SPDX-License-Identifier: Apache-2.0` + `// Copyright (c) 2026 Manu Lorente / kubelab`.
4. Add pre-commit hook checking SPDX header presence on staged source files.
5. Add `go-licenses report` / `pip-licenses --format=markdown` to CI to detect dependency-license drift; fail build on AGPL/GPL/SSPL/BSL dependencies entering the tree.
6. Document the license + contribution model in `README.md`'s "License" section with one paragraph.

## Re-evaluation triggers

Revisit this decision if:
- Pricing model shifts to SaaS multi-tenant (would make AGPL or BSL relevant again).
- A paid Pro tier with materially valuable features is concretely planned (adds dual-license consideration).
- A large client requires a specific license posture as part of procurement (rare, but possible in regulated industries — could justify dual-license).
- iris accepts external contributors at scale (would justify CLA framework, possibly tied to dual-license).
