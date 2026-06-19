---
id: threat-model-agentic-stack
type: security-review
status: draft
created: "2026-06-18"
---

# iris — Layered Threat Model (MAESTRO)

> **Status:** Draft, 2026-06-18 — seeded by an external reference, grounded in the as-built code.
> **Reference:** M. Fontanilla, *"Dissecting the Agentic AI Stack: Threats Layer by Layer"* (kubesandclouds.com / AWS Summit DEV305), which adopts the Cloud Security Alliance **MAESTRO** 7-layer model. fae-brain's `adr-003-security-threat-model-maestro.md` already cites the same deck.
> **Scope:** the iris agent-factory. A companion review of fae-brain lives in the knowledge vault (`10_projects/fae-brain/30-architecture/threat-model-deck-review.md`).
> **Headline:** one **Critical** finding — in local/Docker-host mode the worktree is a *merge-conflict* boundary, not a *security* boundary, and inference credentials are passed into the code-executing process.

## Reference taxonomy (MAESTRO)

| Layer | Name | Scope |
|---|---|---|
| L1 | Foundation Models | LLMs **and** training data |
| L2 | Data Operations | Vector DB, RAG, memory, embeddings |
| L3 | Agent Frameworks | SDKs, reasoning, workflows |
| L4 | Deployment Infrastructure | Containers, networks, orchestration |
| L5 | Evaluation & Observability | Audit, logging, behaviour monitoring, HITL |
| L6 | Security & Compliance | IAM, guardrails, compliance |
| L7 | Agent Ecosystem | Protocols, discovery, A2A, governance |

**Threat catalog (deck, slide 20):** Agent Goal Hijack · Tool Misuse · Identity & Privilege Abuse · Unexpected Code Execution · Context & Memory Poisoning · Insecure inter-agent (A2A) Communication · Cascading failures · Rogue Agents · Overwhelming HITL · Resource overload · Supply Chain. The deck's load-bearing cross-cutting control is a **proxy/gateway** (LiteLLM-shaped) doing egress allow-listing, access-control granularity, rate limiting, budgets, guardrails, MCP gateway, and observability.

## How iris maps

A GitHub issue labelled `iris` → n8n trigger → Go motor dispatches a **pi** worker (TS coding agent, driven as a subprocess by a Go worker per ADR-009) **in a git worktree**; inference via `pi-ai` over OpenRouter/NaN/Ollama (never Claude in the fleet); NATS JetStream bus; a two-stage QA verification gate (coder self-check + independent pi QA + human sign-off); deployed on K3s behind Traefik + Authelia.

**Ground-truth facts that drive the findings (not generic):**
- `internal/worker/driver.go` runs `pi --mode json <prompt>` with `cmd.Env = d.Env` and `cmd.Dir = worktree` → **inference credentials are in the env of the process that executes attacker-influenced code.**
- `internal/worker/worktree.go` isolates *files* per job but runs pi as the worker's own OS user, **no container by default** in local/Docker-host mode.
- `internal/runtime/capability_network.go` — `NetworkManager` is an **opt-in capability, not implemented by local-process** → deny-all egress is not the default.
- `internal/protocol/subjects.go` + ADR-002 — NATS isolation is **per-team connection only, no subject ACLs in v0**.
- `internal/worker/dispatch.go` — the issue **title + body become pi's prompt verbatim** (`Prompt: p.Text`).

## Per-layer findings

### 🔴 CRITICAL — Worktree ≠ security boundary (L4, Unexpected Code Execution)
pi executes shell/builds/tests inside the worktree. ADR-007 is explicit: `worktree = merge-conflict boundary`, orthogonal to `container = security boundary` ("only when on K3s"). So in **local-process / single-Docker-host** delivery — the *primary* consultoría substrate per ADR-002 — there is **no container security boundary by default**: prompt-injected or buggy generated code → host compromise, lateral movement, secret theft. **Likelihood High / Impact Critical.**
**Mitigation:** make the container the *mandatory* execution boundary on **every** substrate (per-job container / gVisor / Firecracker, not just a worktree); default **deny-all egress** (implement `NetworkManager` for the Docker substrate); non-root, no `CAP_*`, read-only rootfs except the worktree; mount only the target repo (never the host FS or the Docker socket); CPU/mem/pids/disk quotas.

### 🟠 HIGH — Inference creds in the code-executing process (L1/L6, Info-disclosure)
`cmd.Env = d.Env` puts OpenRouter/NaN keys in pi's env; injected code can read and exfiltrate them. Also denial-of-wallet: an injected task can burn tokens unbounded.
**Mitigation:** agents never hold upstream provider keys — proxy inference through a **localhost broker (LiteLLM, see ADR-010)** with a per-job scoped token; for IP-sensitive clients pin inference to in-perimeter NaN/Ollama and forbid OpenRouter by config; per-team budgets + rate limits at the proxy.

### 🟠 HIGH — Issue text as goal-hijack vector (L7/L3, Agent Goal Hijack)
The `iris` label is the authorization-to-spawn primitive and the issue body becomes the prompt verbatim. Anyone who can label (or whose untrusted text reaches a labelled issue) achieves goal hijack — on a public repo, an unauthenticated path toward code execution.
**Mitigation:** gate dispatch on **label-applied-by-a-write-access actor** (GitHub App check), not label-present; repo allowlist; wrap the issue body as **delimited untrusted data** in pi's prompt and lock the standing goal in worker config (deck L3).

### 🟠 HIGH — Supply chain: pi + agent-installed deps (L3)
pi is npm-only → Node base image (larger surface than the motor's distroless). Two vectors: pi's own transitive npm tree, and **agent-installed deps** (the agent runs `npm/pip install` of arbitrary packages mid-task — typosquat, malicious post-install, dependency confusion).
**Mitigation:** pin pi by digest, `--ignore-scripts` (already per ADR-009), SBOM + Socket/depscore scan the worker image in CI; route agent installs through an **egress-restricted package mirror** with an allowlist; the QA gate must diff lockfile changes.

### 🟠 HIGH — QA gate subversion (L5)
The verification gate is the moat, hence a target: (a) the coder writes its own self-check and *touches* the tests — a capable model can make a weakened test look green; (b) the QA pi reads attacker-influenced diff/PR text → prompt-injectable verdict; (c) HITL fatigue under PR volume.
**Mitigation:** machine-enforce the anti-cheat rule — CI computes coverage/mutation-score deltas and **blocks** test weakening independently of any agent; mandatory human review on any test-file diff; run the QA pi on a **distinct provider** from the coder with the diff as untrusted data; cap concurrency; surface a per-PR risk score.

### 🟡 MED — NATS bus (L7, pre-multi-tenant)
v0 has no subject ACLs; a motor bug or compromised worker with the NATS endpoint could publish into another team's subjects or subscribe to `team.*.activity`. Single-tenant deploy keeps this Low today; the ADR's "one iris per customer" clause is the live control.
**Mitigation:** before any shared-NATS multi-tenant, add **subject ACLs + per-team nkeys/JWT** and reject envelopes whose `team_id` ≠ the connection's authorized team; pen-test (ADR-002 precondition for removing the clause).

### 🟡 MED — Post-actions & merge (L5)
The PostAction dispatcher fires outbound HTTP on outcomes — an SSRF/egress channel if a binding URL/payload is influenced by issue/PR content. "Never auto-merge" is policy, not code-enforced.
**Mitigation:** allowlist PostAction destination hosts; never interpolate untrusted text into PostAction URLs; enforce **GitHub branch protection** (required review + status checks, no force-push) so the no-auto-merge rule is server-side.

### 🟡 MED — K3s / Authelia / secrets (L4/L6)
Human sign-off is gated by Authelia forwardAuth → Authelia is the trust root for the only non-agent control; its compromise = unsupervised merges. The Docker socket mounted into a worker = instant host root.
**Mitigation:** harden Authelia (MFA on the sign-off route, short sessions); secrets in SOPS/Vault, short-lived task-scoped (deck L6); **never** mount the Docker socket into a worker; K8s NetworkPolicy default-deny for worker pods; non-root pods, seccomp/AppArmor, read-only rootfs.

## Next actions (→ tracked as the iris security epic)

1. **IRIS-SEC-1 (P0):** mandatory per-job execution container on all substrates + `NetworkManager` Docker path with default deny-all egress. *Closes the Critical.*
2. **IRIS-SEC-2:** remove provider keys from the pi env — worker proxies inference (localhost + per-job scoped token); land with ADR-010 / supersedes SDD-034i (#13).
3. **IRIS-SEC-3:** dispatch authorization (trusted-labeller GitHub App check + repo allowlist; issue body as delimited untrusted data).
4. **IRIS-SEC-4:** machine-enforced anti-cheat in the QA gate (CI coverage/mutation deltas; QA pi on a distinct provider; diff as untrusted).
5. **IRIS-SEC-5:** supply-chain CI (pin pi by digest, SBOM + Socket/depscore, allowlisted package mirror; branch protection).
6. **IRIS-SEC-6 (pre-multi-tenant):** NATS subject ACLs + per-team nkeys/JWT + envelope `team_id` enforcement, pen-tested.

## References
- M. Fontanilla, *Dissecting the Agentic AI Stack: Threats Layer by Layer* (kubesandclouds.com/en/downloads/).
- CSA MAESTRO; OWASP GenAI Top-10 for Agentic Applications.
- iris: `docs/adr/adr-002`, `adr-007`, `adr-009`; `internal/worker/{driver,worktree,dispatch}.go`; `internal/runtime/capability_network.go`; `internal/protocol/subjects.go`.
- fae-brain companion: `knowledge/10_projects/fae-brain/30-architecture/threat-model-deck-review.md`.
