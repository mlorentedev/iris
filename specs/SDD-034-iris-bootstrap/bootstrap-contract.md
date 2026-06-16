---
id: "SDD-034-iris-bootstrap-bootstrap"
type: bootstrap-contract
status: accepted
created: "2026-05-17"
accepted: "2026-05-17"
tags: [spec, bootstrap, contract, iris]
template_version: "1.0"
---

# SDD-034: Iris bootstrap — Bootstrap Contract

> **Naming**: lives at `<repo>/specs/<feature-id>/bootstrap-contract.md`. Optional artifact — present here because iris is a NEW substrate (greenfield Go motor + workers + console).

> **Normative, not descriptive.** Each section below is a promise: a peer with this doc + repo at HEAD on a clean machine must reach a passing smoke test. If any section cannot be honored yet, the substrate is not bootstrappable — DO NOT proceed to `tasks.md`.

## 1. Runnable environment

```bash
make dev
```

**Substrate constraint:** dev runs **Local-process** per [adr-002-orchestrator-architecture](../../docs/adr/adr-002-orchestrator-architecture.md) line 183 — motor as native binary, workers + NATS in containers. Motor does NOT run in Docker during development (fast iteration loop via `air` hot-reload).

**Prerequisites:**

- Go 1.26+ (`go version` returns `go1.26.0` or higher) — ✓ installed
- Python 3.12+ + Poetry — ✓ installed (matches kubelab toolkit setup)
- Docker Engine 28+ for worker + NATS containers — ✓ installed
- `air` CLI for Go hot-reload: `go install github.com/air-verse/air@latest` — needs install (Makefile target `make install-tools`)
- Tailwind standalone binary — downloaded by `make install-tools` (~50 MB, cached after first run)

**Ports (loopback only — no Traefik in dev):**

- `:8080` — motor HTTP API + HTMX console (CSS served from `embed.FS`)
- `:4223` — NATS JetStream client port (intentionally NOT default `:4222` to avoid clash if kubelab adds NATS later)

**What `make dev` does (verbatim sequence):**

1. Pre-flight: verify Go ≥1.26, `air` installed, `tailwindcss` binary present (fail-fast with pattern-agent-oriented-errors three-element format)
2. `docker compose -f compose.dev.yml up -d nats worker-pi`
3. Wait for NATS health: `nats --server localhost:4223 server check connection` (5s timeout, retry 3x)
4. `tailwindcss --watch -i input.css -o internal/console/static/app.css` (background process)
5. `air -c .air.toml` (motor binary; hot-reload on `.go` and template changes)

**Production target (out-of-scope for this contract, tracked in SDD-034g):** Kubernetes deploy in Manu's own kubelab K3s. iris is dogfooded as Manu's own first customer before any external consultoría delivery. Image artifacts produced by SDD-034a (multi-stage Dockerfile + K8s-readiness primitives) are the bridge to that deploy.

**Opt-in extensions (not in `make dev` default):**

- Ollama runtime: `make dev FEATURES=ollama` (adds compose override starting `ollama` container on `:11434`)
- Qdrant vector store: `make dev FEATURES=qdrant` (adds compose override starting `qdrant` container on `:6333`)
- Both: `make dev FEATURES=ollama,qdrant`

## 2. Passing smoke test

```bash
make smoke-test
```

**Prerequisite:** `make dev` running in background (substrate up). Without it, the 5 checks fail-fast with informative curl/dial timeouts.

**5 atomic checks (ordered — each depends on previous; first non-zero exit halts):**

| # | Invariant | Command |
|---|---|---|
| 1 | Motor HTTP up | `curl -fsS http://localhost:8080/healthz` (200, body `{"status":"ok"}`) |
| 2 | Motor ready (DB migrated, NATS connected) | `curl -fsS http://localhost:8080/readyz` (200, body `{"db":"ok","nats":"ok"}`) |
| 3 | NATS reachable from host | `nats --server localhost:4223 server check connection` |
| 4 | Worker pi container running | `docker compose ps -q worker-pi \| xargs docker inspect --format '{{.State.Status}}' \| grep -q '^running$'` |
| 5 | End-to-end NATS pub/sub roundtrip | `nats --server localhost:4223 pub iris.smoke "ping" && timeout 3 nats --server localhost:4223 sub iris.smoke --count 1 \| grep -q "ping"` |

**Why these 5 and not others:**

- 1 + 2 separate "binary runs" (`/healthz`) from "binary READY with deps connected" (`/readyz`). Industry standard for K8s liveness vs readiness probes; aligns with container-workflow § health checks.
- 3 validates NATS independently (catches port conflict with kubelab if its NATS is running on `:4222`).
- 4 validates worker substrate. Without worker, motor is "ok" but system can't do anything — would be a false-positive without this check.
- 5 is the ONLY end-to-end check that proves motor + NATS + worker understand each other semantically. The previous 4 can pass with a broken-by-config system; this catches it.

**Pre-flight for `make smoke-test`:**

- Requires `nats` CLI on host: `go install github.com/nats-io/natscli/cmd/nats@latest` (added to `make install-tools`)
- Output on success: exit 0, line `[smoke] 5/5 checks passed` to stdout
- Output on failure: exit non-zero, line `[smoke] failed at check N: <description>` to stderr (three-element format per pattern-agent-oriented-errors)

**Strategic alignment:** this `make smoke-test` is **shape-identical to the CI smoke test for SDD-034g** (kubelab K3s deploy). Only URLs change (`http://localhost:8080` → `https://iris.kubelab.live`). The smoke-test becomes the single "iris-is-up" contract spanning dev + Manu's kubelab production + future consultoría customer environments.

## 3. Contract document

**Approach:** code is the contract (Go interfaces + structs as primary source of truth) — same shape as agent_crew's production-validated model, but with the gaps they left open closed from day 1. See agent_crew analysis § 6, 7, 9 for the prior-art audit.

### Surface table

| Surface | Lives in | Source-of-truth artifact | Stability |
|---|---|---|---|
| NATS protocol envelope (7+ message types) | `internal/protocol/messages.go` | Go discriminated-union struct | **FROZEN** |
| NATS subject taxonomy + validator | `internal/protocol/subjects.go` | `team.<id>.<channel>` regex validator | **FROZEN** |
| Runtime abstraction | `internal/runtime/runtime.go` | `AgentRuntime` Go interface + capability sub-interfaces (`OllamaManager`, `QdrantManager`, `NetworkManager`, `SidecarManager`) | **FROZEN** (base) / **APPEND-ONLY** (capability set grows additively) |
| HTTP API | `internal/api/routes.go` | `chi` router + **OpenAPI 3.1 generated to `docs/api.yaml`** via `huma` or `swaggo/swag` (decision in SDD-034a tasks.md) | **BETA** under `/api/v1/*` prefix; v2 may run in parallel before v1 deprecates |
| Console HTMX URLs | `internal/console/handlers/*.go` | Route registrations + HTML templates (`embed.FS`) | **BETA** — UI evolves, URLs stable within `/v1`. Per [adr-001-frontend-htmx-go](../../docs/adr/adr-001-frontend-htmx-go.md) |
| Error format | every error boundary | pattern-agent-oriented-errors three-element form (ERROR / WHY / FIX) | **FROZEN** |
| DB schema | `internal/db/migrations/` | `golang-migrate` numbered up/down files, NEVER edited after merge | **APPEND-ONLY** — every change is a new migration, never an edit |
| Data model | `internal/db/queries/*.sql` + `internal/db/sqlc.go` | `sqlc` codegen from SQL (NOT GORM, NOT AutoMigrate) | Same lifecycle as migrations |
| Auth providers | `internal/auth/` | `Provider` interface; v0 implements `local` (cookie+CSRF+bcrypt) only. `noop`/`oidc`/`auth0` declared as TODO, NOT shipped | **BETA** (extension point; v0 ships ONLY `local` — `noop`/`oidc`/`auth0` are gaps closed in dedicated SDDs, never half-implemented like agent_crew) |
| Multi-tenant discriminator | every persisted model | `team_id` column + middleware injection per [adr-002-orchestrator-architecture](../../docs/adr/adr-002-orchestrator-architecture.md) | **FROZEN** |
| Feature gating | `features.json` (sibling to `tasks.md`) | Per pattern-feature-list-as-primitive — pass-state irreversible via harness | **FROZEN** schema; entries grow with implementation |

### Stability tier definitions (explicit — what agent_crew lacks)

- **FROZEN** — breaking changes require an ADR, a deprecation window of at least one minor release, and a migration note in `90-lessons.md`. Bug fixes welcome.
- **BETA** — may change between SDD-034a and SDD-034f without ADR, **provided** that (a) the change is captured in `features.json` (so smoke-test mirrors the new shape), (b) the `OpenAPI v3.1` doc is regenerated and committed in the same PR, (c) a one-line CHANGELOG entry is added.
- **APPEND-ONLY** — past entries are immutable. Forward changes are net-new entries. Applies to DB migrations and to `features.json` (a `state: passing` entry never reverts to `state: pending` without a brand-new entry and explicit rationale).

### Gap closures vs agent_crew prior art

| agent_crew gap (per analysis § 9) | iris closure (day-1) |
|---|---|
| No OpenAPI / API reference | `make api-docs` regenerates `docs/api.yaml` from chi handlers + struct tags; CI fails if drift detected between generated spec and committed file |
| No `v1/` prefix on routes | All HTTP routes namespaced under `/api/v1/*` from first commit; `v2` paralelo policy documented when needed |
| No stability markers | This very table — FROZEN/BETA/APPEND-ONLY explicit per surface |
| GORM `AutoMigrate` (data loss risk) | `golang-migrate` numbered migrations only; `AutoMigrate` BANNED by lint rule + PR review |
| No deprecation pattern | `BETA` tier definition above mandates deprecation window for breakages |
| Frontend types manually mirror backend | iris uses HTMX server-side rendering — there is no separate client type system to drift. Side-effect of [adr-001-frontend-htmx-go](../../docs/adr/adr-001-frontend-htmx-go.md) |
| Auth providers declared but not implemented (`oidc`/`auth0`) | v0 ships ONLY `local`. `noop`/`oidc`/`auth0` are explicit GAPS closed in dedicated SDDs (NOT half-implemented stubs) |
| `OrgID` discriminator without NATS subject ACLs | iris ships `team_id` + per-team NATS connection with subject ACL enforcement post-MVP (ADR-038 § "Multi-tenant boundary v0") |

### Provider abstraction (permanent constraint)

Two orthogonal axes (per ADR-007, refining ADR-002's flat enum):

- **Runtime / harness:** `pi` (v0 fleet) | `opencode` | `hermes` (supervision plane, future — [adr-002-orchestrator-architecture](../../docs/adr/adr-002-orchestrator-architecture.md)).
- **Inference (via `pi-ai`):** `OpenRouter` | `NaN` | `Ollama` | `Anthropic-API` (only if a client brings their own key) — **never Claude-Code-OAuth in the fleet** (Anthropic ToS).

`ollama` is an inference provider, not a runtime. The motor under `internal/` must contain ZERO provider-specific code; provider-specific logic lives only in capability sub-interface implementations (e.g., `internal/runtime/pi/`).

### Features.json IDs (to be populated in Q4)

This contract will reference feature IDs once Q4 (task breakdown) is captured. Each task in the breakdown maps to one or more `features.json` entries with executable verification commands.

## 4. Task breakdown

5 atomic tasks, each one PR scope, ordered by dependency. Each verification is the executable command that the harness (manual for now, iris scheduler-verifier post-SDD-034c) runs to transition the matching `features.json` entry from `verifying` to `passing`/`failing`. **Per pattern-feature-list-as-primitive pass-state gating: agent CANNOT mark `passing` — only the harness, on `exit 0`, may.**

| # | Outcome | Verification command |
|---|---|---|
| 1 | **Repo plumbing** — LICENSE Apache 2.0 + NOTICE + SPDX pre-commit hook + `.gitignore` Go-style + minimal README pointing at this spec + `go mod init github.com/mlorentedev/iris` (placeholder until D40) + `.golangci.yml` (boring ruleset) | `pre-commit run --all-files && golangci-lint version && go mod verify` |
| 2 | **Motor skeleton** — `internal/{api,scheduler,runtime,protocol,post-actions,console,db}/` with stub `doc.go` per package + `cmd/motor/main.go` with `chi` router exposing `/healthz` + `/readyz` + 12-factor env config (`envconfig` or stdlib) + structured logging (slog handler, logfmt) + SIGTERM graceful shutdown 15s timeout | `go build ./... && go vet ./... && ./bin/motor & sleep 1; curl -fsS localhost:8080/healthz && curl -fsS localhost:8080/readyz; kill %1` |
| 3 | **DB layer** — `sqlc.yaml` + `internal/db/migrations/0001_init.{up,down}.sql` (minimal schema: 1 `teams` table with `team_id` + audit cols) + `internal/db/queries/teams.sql` (1 select + 1 insert) + `sqlc generate` produces `internal/db/sqlc/` Go types + `golang-migrate` integration in `cmd/motor` (run migrations on startup) | `make migrate-up && sqlc generate && go test ./internal/db/... -count=1` |
| 4 | **Dev environment** — `Makefile` targets (`dev / install-tools / smoke-test / docker-build / api-docs`) + `.air.toml` for Go hot-reload + `compose.dev.yml` with services `nats` (port 4223) + `worker-pi` (stub image, healthcheck only for now) + `make smoke-test` script implementing the 5 checks from § 2 | `make install-tools && make dev & sleep 5; make smoke-test; kill %1` |
| 5 | **Production Dockerfile + OpenAPI generation** — `Dockerfile` multi-stage (`golang:1.26-alpine` build → `gcr.io/distroless/static:nonroot` runtime, SPDX header, non-root user) + `make docker-build` produces local `iris:dev` tag + `huma` or `swaggo/swag` integration generates `docs/api.yaml` from chi routes via `make api-docs` + CI rule rejects drift between generated and committed `docs/api.yaml` | `make docker-build && docker run --rm iris:dev /motor --version \| grep -q '0.1.0' && make api-docs && git diff --exit-code docs/api.yaml` |

**Justification of splits:**

- **1 before 2** — pre-commit + LICENSE prevent non-SPDX commits from the first real code
- **2 before 3** — motor binary runs without DB (`/healthz` returns 200 without `/readyz`); separates "binary boots" from "stack ready"
- **3 before 4** — `make dev` requires migrations on startup; without 3, motor in 4 crashes
- **4 before 5** — multi-stage Dockerfile needs a compilable binary (task 2) + existing make targets (task 4)
- **5 closes K8s-readiness** — Dockerfile + OpenAPI are the two artifacts SDD-034g (kubelab K3s deploy) consumes downstream

**Sizing:** 5 tasks, each ≤2h → ~5-7h total. Matches the expanded estimate in `iris/11-tasks.md` SDD-034a after K8s-readiness primitives were added.

**Side effect:** these 5 verifications populate `features.json` (sibling to `tasks.md`) with 5 initial entries, all `state: pending`. The harness flow (manual for now, iris scheduler-verifier post-SDD-034c) is the only path to `state: passing`.

## Acceptance signal

A peer with a clean machine can:

1. Clone the repo.
2. Read this doc only.
3. Run section 1's command → environment up.
4. Run section 2's command → exit 0.

## References

- Pattern: pattern-spec-driven-development
- Vault backlog: `10_projects/iris/11-tasks.md` (SDD-034a)
- Architecture ADR: [adr-002-orchestrator-architecture](../../docs/adr/adr-002-orchestrator-architecture.md) (renumbered from kubelab/adr-038 on 2026-05-19)
- License ADR: [adr-003-license-apache-2](../../docs/adr/adr-003-license-apache-2.md) (renumbered from kubelab/adr-040 on 2026-05-19)
- Frontend ADR: [adr-001-frontend-htmx-go](../../docs/adr/adr-001-frontend-htmx-go.md) (renumbered from kubelab/adr-035 on 2026-05-19)
- Positioning: vault strategy reports (not in the repo, added 2026-05-21)
- Architectural boundary ADR (added 2026-05-21): [adr-005-architectural-boundary](../../docs/adr/adr-005-architectural-boundary.md)
