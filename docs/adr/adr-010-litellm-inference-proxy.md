---
id: adr-010-litellm-inference-proxy
type: adr
status: proposed
created: "2026-06-18"
---

# ADR-010 — LiteLLM as the iris inference proxy

> **Status:** Proposed, 2026-06-18
> **Supersedes:** SDD-034i (iris#13) — the inference-proxy seam that ADR-007 placed and deferred
> **Refines:** [adr-007-iris-v0-coding-loop](./adr-007-iris-v0-coding-loop.md) (§"Runtime × Inference axes"), [adr-005-architectural-boundary](./adr-005-architectural-boundary.md) (§"iris EDGE" — LiteLLM already listed as adopted substrate)
> **Tracking:** mlorentedev/iris#29 · mlorentedev/iris#32 (SEC-001 / IRIS-SEC-2)

## Decision

Adopt self-hosted **LiteLLM** (Apache-2.0) as the iris inference proxy. pi workers talk to a `localhost:4000` (or internal-network) LiteLLM endpoint using a **per-job scoped token**; they never hold an upstream provider key. The motor issues the scoped token at job dispatch time and revokes it when the job completes. Upstream provider keys (OpenRouter, NaN, Ollama) live exclusively in the motor's credential store — never in the worker process environment.

This supersedes SDD-034i, which identified the seam but left the implementation open.

## Context

ADR-007 separated the inference axis from the harness axis and explicitly deferred the "intelligent inference proxy (cost/availability routing + PII redaction + budgeting)" to SDD-034i (iris#13), noting: *"the seam is placed here, full implementation deferred."* ADR-005 listed LiteLLM as an already-decided EDGE substrate.

Two forces now make the decision urgent rather than deferrable.

**Force 1 — Security finding from the v0 threat model.**
The current implementation passes the full worker process environment into every pi subprocess. In `cmd/worker/main.go` (line 96) `w.Env` is set to `os.Environ()`. In `internal/worker/worker.go` (line 124) `driverFor` forwards `w.Env` into `Driver{Env: w.Env}`. In `internal/worker/driver.go` (line 56) `cmd.Env = d.Env` applies that environment to the `exec.CommandContext` that spawns pi.

pi drives arbitrary tool calls — shell commands, file reads, HTTP requests — inside a git worktree. Any secret visible in pi's environment is reachable from within the coding session. Provider keys (`OPENROUTER_API_KEY`, `NAN_API_KEY`, etc.) must therefore not be in that environment. The current design violates this directly. (Full analysis: `docs/security/threat-model-agentic-stack.md`.)

**Force 2 — Provider routing, cost-tracking, and failover are unsolved.**
ADR-007 lists inference switchability (OpenRouter / NaN / Ollama) as a config concern, not an architecture concern. In practice, switching inference backend requires redeploying workers with different environment variables. There is no per-team budget enforcement, no provider health check, no cost attribution, and no rate-limit handling — all capabilities that a proxy layer provides for free once the seam is crossed.

**What iris is NOT delegating to LiteLLM.**
The boundary matters. LiteLLM owns: provider routing, rate-limit/retry, load-balancing, provider health, cost-tracking metadata, and the credential store for upstream keys. iris still owns: per-team budget enforcement (the motor checks LiteLLM's spend API before dispatch), scheduling, worktree isolation, the verification gate, NATS protocol, and distributed tracing (`TraceID` on every envelope maps to an OTel trace forwarded through LiteLLM's headers).

**The `claude` constraint is unchanged.** ADR-002's permanent prohibition on Claude in the fleet applies to LiteLLM's routing table: the `claude/*` and `claude-code/*` providers must not be configured in the fleet LiteLLM instance. This is enforced by policy in the LiteLLM config file, not by software capability.

## Decision drivers

- The worker process executes code written by a model. Inference provider keys in that process's environment are one tool call away from exfiltration — a straightforward secret-in-subprocess threat that LiteLLM's localhost-proxy pattern eliminates.
- LiteLLM is already listed as an EDGE substrate in ADR-005 (§ EDGE table). Adopting it is executing a prior decision, not adding a new dependency.
- Apache-2.0 license is compatible with iris's Apache-2.0 (ADR-003).
- LiteLLM provides OpenAI-compatible endpoints: pi's `pi-ai` layer already speaks OpenAI-compatible APIs, so the worker integration requires only a base-URL and token swap — no pi fork or custom plugin.
- Per-job scoped tokens enable per-job spend attribution and revocation, which is the correct granularity for a coding-agent system where one runaway job should not exhaust the team's quota.
- Provider failover (OpenRouter → NaN → Ollama-local) becomes a LiteLLM config change, not a worker redeployment.

## Consequences

**Positive:**
- Provider keys are fully removed from worker container environments. The worker holds only a short-lived scoped token for its own job, issued by the motor.
- `internal/worker/driver.go` `cmd.Env = d.Env` becomes safe: the env that reaches pi contains `OPENAI_BASE_URL=http://litellm:4000/v1` and `OPENAI_API_KEY=<scoped-token>` — no upstream secrets.
- Cost telemetry and provider health are operational concerns delegated to LiteLLM's dashboard, not instrumented in iris code.
- Switching inference backend (OpenRouter → NaN, or adding Ollama as a fallback) is a LiteLLM config change; zero worker redeployments.
- Per-team budget caps are implementable: motor calls LiteLLM's `/spend/check` before emitting the `user_message` envelope; over-budget teams get a `system_command` rejection rather than a runaway job.

**Negative / constraints:**
- LiteLLM becomes a required runtime dependency alongside NATS. The `compose.dev.yml` and Kustomize overlays need a LiteLLM container. Docker substrate deployments add one more container to the team stack.
- The motor must implement token issuance (short-lived JWTs or LiteLLM virtual-key API) at job dispatch. This is new motor logic that lands in the scheduler package (currently a placeholder at `internal/scheduler/doc.go`).
- In Ollama-only local-dev mode, LiteLLM is still required to sit in front of the Ollama endpoint, adding a hop that adds latency (single-digit ms; acceptable for a coding loop whose turns take seconds to minutes).

**Follow-up sweeps required:**
- `cmd/worker/main.go`: remove `Env: os.Environ()` as the default; replace with a filtered env that strips any `*_API_KEY` and `*_SECRET` variables and injects only `OPENAI_BASE_URL` + the per-job scoped token.
- `internal/worker/worker.go` / `internal/worker/driver.go`: document that `Driver.Env` must never contain upstream provider keys — add a comment and a startup validation check.
- `internal/scheduler/` (placeholder): implement token issuance and revocation as part of the job dispatch lifecycle.
- `internal/runtime/runtime.go` comment at line 80: update to note that inference is no longer configured via `AgentSpec.Env` directly — it flows through LiteLLM.
- `compose.dev.yml`: add the LiteLLM service with a dev config that routes to Ollama by default.
- `docs/adr/adr-007-iris-v0-coding-loop.md` §"Runtime × Inference axes": add a forward reference to this ADR as the implementation of the SDD-034i seam.

## Alternatives considered

**Route pi directly to provider (current state).**
Already ruled out by the threat model finding above. The worker process executes model-generated code; any secret in its environment is reachable from that code. This is the threat that must be closed before v0 goes beyond dogfood.

**Per-worker env file with provider keys (without a proxy).**
Reduces the blast radius slightly (key not in every subprocess) but does not eliminate it: the worker process itself holds the key, and pi still spawns as a subprocess of the worker with env inheritance. The key is still one `os.Getenv` call away from exfiltration via a tool call.

**Custom Go reverse proxy instead of LiteLLM.**
Would let iris own the credential-proxy logic entirely, avoiding the LiteLLM runtime dependency. Rejected: LiteLLM covers provider routing, rate limiting, spend tracking, and model failover — a substantial capability surface. Building this in Go is ≥4 weeks of effort for functionality LiteLLM provides today under Apache-2.0. ADR-005's adoption criterion ("no sense building a LiteLLM if one already exists") applies directly.

**Defer until a compliance customer asks.**
The threat is present in the current implementation regardless of whether a compliance customer asks. The cost of the fix (base-URL + token swap in the worker; one container in the stack) is low. Deferral would mean shipping a known credential-exposure path into any consultoría engagement.
