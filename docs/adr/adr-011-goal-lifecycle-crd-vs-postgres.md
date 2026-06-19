---
id: adr-011-goal-lifecycle-crd-vs-postgres
type: adr
status: proposed
created: "2026-06-18"
---

# ADR-011 — Goal lifecycle: Postgres-loop with a controller-runtime-shaped reconciler interface

> **Status:** Proposed, 2026-06-18
> **Supersedes:** —
> **Refines:** [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md) (§"Scheduler" component, deferred in the "What's NOT decided" table), [adr-007-iris-v0-coding-loop](./adr-007-iris-v0-coding-loop.md) (§"The loop" — issue → dispatch → pi → gate → sign-off lifecycle)
> **Tracking:** mlorentedev/iris#29

## Decision

Implement Goal lifecycle using a **Postgres-backed reconciliation loop** driven by the Go motor, not as a Kubernetes CRD. The scheduler package (`internal/scheduler/`, currently a placeholder) exposes a `GoalReconciler` interface shaped after `sigs.k8s.io/controller-runtime`'s `Reconciler` — a single `Reconcile(ctx, Request) (Result, error)` method — so the internal contract is controller-runtime-compatible without the external CRD dependency. A future CRD adapter can implement the same interface and swap in without breaking callers.

v0 uses SQLite (the existing ADR-002 persistence decision); the design is Postgres-ready for kubelab deployment (ADR-007 explicitly targets kubelab's PostgreSQL).

## Context

iris#29 identifies the Goal lifecycle as a fork that must be decided explicitly. Two credible options exist:

**Option A — Goal-as-CRD + controller-runtime reconciler.** Model `Goal` as a Custom Resource (e.g. `goals.iris.mlorentedev.io/v1alpha1`). The motor runs a controller-runtime manager that watches Goal CRs, reconciles them through the issue → pi-dispatch → verification-gate → sign-off → merge lifecycle, and reports status via CR conditions. This is the "iris is Kubernetes for agents" framing.

**Option B — Bespoke Postgres loop.** The motor holds Goals in a SQL table, runs a periodic poll (or LISTEN/NOTIFY) loop, and drives state transitions through the same lifecycle using an in-process state machine.

**Why the fork is non-trivial.** `sigs.k8s.io/controller-runtime v0.23.3` is already in `go.mod` (line 21) and is actively used in `internal/runtime/k8s/k8s.go` (the K8s substrate adapter imports `ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"`). The dependency exists. The question is whether to use controller-runtime's manager/reconciler machinery to drive Goal lifecycle, not whether to add the dependency.

**The consultoría substrate constraint.**
ADR-002 is unambiguous: *"Docker | single-host deployments — primary consultoría delivery model."* The K8s substrate is opt-in; Docker is the primary delivery vehicle. A Goal-as-CRD design makes the motor's core loop a hard Kubernetes dependency: without a cluster, `kubectl apply` of a Goal CR has nowhere to land, and the motor's reconciler has nothing to watch. This immediately breaks every Docker and local-process deployment — which are the two substrates iris uses most (consultoría Docker + Manu's local dev). The K8s runtime adapter already handles this split cleanly: `internal/runtime/k8s/` is only loaded when the K8s substrate is selected; the Docker and local adapters need no cluster. A CRD-backed Goal lifecycle would collapse that clean separation.

**What controller-runtime already does in this repo.**
The existing use in `internal/runtime/k8s/k8s.go` is purely as a typed client for Sandbox CRs (agent-sandbox substrate). It does not run a controller-runtime manager or use the reconciler loop. The motor currently has no manager or informer cache. Adding a full controller-runtime manager to drive Goal lifecycle in Docker deployments would pull in the informer/cache/webhook machinery for no benefit — those subsystems exist to watch Kubernetes API objects, not Postgres rows.

**The interface insight.**
controller-runtime's `Reconciler` interface is `Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error)`. This is not a K8s-specific contract; it is a clean idempotent-reconciliation pattern. iris can adopt the pattern without adopting the manager. A `GoalReconciler` interface in `internal/scheduler/` that mirrors this signature gives the same structural benefit — testability, pluggability, "tell me what changed and I'll reconcile" semantics — without requiring a CRD or a running cluster. If a K8s-native deployment is ever required, a `CRDGoalReconciler` adapter that bridges Goal CRs to the same interface is a bounded implementation task, not a re-architecture.

**The scheduler is still a placeholder.**
`internal/scheduler/doc.go` is a single-line package comment. No design is locked in. This is the correct moment to decide.

## Decision drivers

- The primary deployment substrate is Docker (ADR-002 Component 4, §"Substrates supported in v0"). Making the scheduler's core loop a Kubernetes dependency breaks Docker and local deployments. This is the decisive constraint.
- ADR-005 explicitly lists the K8s substrate as opt-in: *"The Docker adapter is NOT a stepping stone for the K8s adapter — it is the primary delivery substrate for v0 consultoría engagements."* The same logic applies here.
- controller-runtime's `Reconciler` interface is substrate-neutral. Adopting the interface shape without the manager gives all the structural benefits (idempotency contract, `Result{RequeueAfter}` for retry backoff, clean error surface) while keeping the implementation portable.
- SQLite (current v0) and Postgres (kubelab, target ADR-007) both support the polling or LISTEN/NOTIFY patterns that drive the loop. No new dependency is introduced.
- Goal state (pending / dispatched / verifying / awaiting-sign-off / merged / failed) maps cleanly onto a SQL enum column. Transitions are transactional. A Postgres loop with `SELECT ... FOR UPDATE SKIP LOCKED` gives exactly-once dispatch semantics that CRD controllers achieve via optimistic concurrency — equivalent power, no cluster required.
- The existing `internal/db/sqlc` pattern (code-generated queries over SQLite, Postgres-compatible SQL) already demonstrates how iris adds persistence without ORM complexity. Goal follows the same pattern: a migration, a sqlc query file, generated Go types.
- A future K8s-native requirement is a non-breaking extension: add a `CRDGoalReconciler` that watches Goal CRs and calls the same `GoalReconciler` interface. The motor's scheduler loop, sign-off logic, and verification gate are untouched.

## Consequences

**Positive:**
- The scheduler works on every substrate (Docker, local, K8s) without a running Kubernetes cluster.
- Goal state is durable in the same SQLite/Postgres store as teams, queryable with the same sqlc tooling. The HTMX console can display Goal status via a standard SQL read — no K8s API client needed in the frontend path.
- The `GoalReconciler` interface is trivially unit-testable: implement a fake, call `Reconcile`, assert the resulting `Result`. No cluster, no fake client-go, no envtest.
- controller-runtime's `Reconcile` signature is familiar to any Go engineer who has written a K8s operator. Naming and shaping the internal interface after it lowers the cognitive cost for contributors.
- The retry/backoff model maps directly: `reconcile.Result{RequeueAfter: d}` is the pattern; the Postgres loop's equivalent is a `next_reconcile_at` column updated on each transition.

**Negative / constraints:**
- The motor owns more lifecycle logic than it would if the K8s control plane drove it. This is a deliberate trade: correctness and portability over elegance.
- LISTEN/NOTIFY (for low-latency Goal pickup) is Postgres-only; SQLite falls back to polling. In v0 on SQLite the polling interval (default: 2s) is acceptable for a coding-agent loop whose jobs take minutes. The scheduler must document this per-store behavior difference.
- If a large multi-tenant deployment ever needs the "GitOps / kubectl apply a Goal" UX, the CRD adapter is an additive task — but the motor must be running Kubernetes at that point anyway, which makes it an explicit future scope item, not a missing feature.

**Follow-up sweeps required:**
- `internal/scheduler/doc.go`: replace the placeholder with the `GoalReconciler` interface, `Request` type, and `Result` type (mirroring controller-runtime's surface, with `import` avoided — iris-native types only).
- Add `internal/db/migrations/0002_goals.up.sql`: the `goals` table with columns for `id`, `team_id`, `status` (enum: `pending | dispatched | verifying | awaiting_sign_off | merged | failed`), `issue_ref`, `pi_job_id`, `next_reconcile_at`, `created_at`, `updated_at`.
- Add `internal/db/queries/goals.sql` and regenerate sqlc.
- Implement `internal/scheduler/postgres_reconciler.go`: the poll-or-LISTEN loop, the state machine, and the `SELECT ... FOR UPDATE SKIP LOCKED` dispatch pattern.
- `internal/scheduler/sqlite_reconciler.go`: the polling fallback for v0 SQLite.
- Wire the scheduler into `cmd/motor/main.go` (currently it has no scheduler start call).
- Document that `IRIS_DB_PATH` (SQLite) uses poll-based scheduling and `DATABASE_URL` (Postgres) enables LISTEN/NOTIFY.

## Alternatives considered

**Goal-as-CRD + controller-runtime manager.**
The "iris is Kubernetes for agents" framing is compelling architecturally and GitOps-aligned. Rejected for v0 for one decisive reason: it hard-couples the motor's core scheduling loop to Kubernetes, breaking Docker and local-process deployments, which are the primary delivery substrate per ADR-002. controller-runtime's manager, informer cache, and webhook machinery are genuinely useful when you have a K8s API server to watch; they add zero value and real complexity when the motor runs as a Docker container or a local binary. The opt-in K8s runtime adapter already demonstrates the correct pattern for separating K8s-specific logic from the substrate-neutral core. Deferred: if iris ever ships a K8s-operator mode as an explicit product tier (not just the K8s runtime substrate adapter), revisit this ADR.

**Temporal or a workflow engine.**
A workflow engine like Temporal would provide durable execution, history, and retry semantics that the Postgres loop has to implement manually. Rejected: the dependency and operational weight (Temporal server, worker SDK) is disproportionate to v0's single-host, single-binary wedge (ADR-002). A Postgres loop with `SELECT ... FOR UPDATE SKIP LOCKED` is the standard Go pattern for this problem size and is consistent with iris's "boring tech" persistence philosophy. Reconsider if the retry/history requirements exceed what SQL can express cleanly.

**Redis-backed queue (no Postgres).**
Would simplify the poll loop but adds a third stateful dependency (NATS + SQLite/Postgres + Redis). The `next_reconcile_at` column in Postgres achieves the same delayed-requeue without a new service. Rejected on dependency-minimization grounds.
