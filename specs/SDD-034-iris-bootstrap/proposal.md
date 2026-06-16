---
id: "SDD-034-iris-bootstrap"
type: spec
status: implementing
created: "2026-05-17"
tags: [spec, proposal, iris, bootstrap]
template_version: "1.0"
vault_id: "SDD-034a"
---

# SDD-034: Iris bootstrap

> **Naming**: file lives at `<repo>/specs/<feature-id>/proposal.md`. Vault cross-reference: `SDD-034a` in `10_projects/iris/11-tasks.md` (and `10_projects/knowledge/11-tasks.md` until full migration).

## Why

<!-- from 10_projects/iris/11-tasks.md SDD-034a + adr-005-architectural-boundary § Consequences SDD-034 stream tasks -->

Iris hoy es una arquitectura aceptada (ADR-002/004/005) pero sin substrate runnable — `~/Projects/iris/` solo tiene specs. Sin Go motor + DB layer + Dockerfile + dev environment ningún SDD-034 posterior puede empezar: SDD-034b NATS no tiene proceso donde correr, SDD-034c K8s runtime adapter (post OQ-1: thin wrapper sobre agent-sandbox) no tiene cliente que orquestar, SDD-034d agents Combo 8 no tienen coordinador al que reportar, SDD-034g kubelab deploy no tiene imagen `iris:dev` que desplegar. SDD-034a es el blocker arquitectural que convierte iris de "ADRs cristalizados" en "código que arranca".

## What

Cinco outputs concretos verificables tras este PR:

1. **Binario `motor` que arranca:** `go build ./cmd/motor` produce binary que sirve HTTP en `:8080` con `/healthz` (liveness) + `/readyz` (readiness) + 12-factor env config + slog logfmt structured logging + SIGTERM graceful shutdown 15s timeout.
2. **DB layer migrado:** SQLite (v0) con migración inicial 0001 creando `teams` table; sqlc-generated typed queries; `make migrate-up` aplica migraciones idempotentemente; motor aplica migrations on startup.
3. **Dev environment ergonomic:** `make dev` levanta NATS:4223 + worker-pi stub + motor via air (hot-reload); `make smoke-test` ejecuta los 5 checks del bootstrap-contract § 2 y exit 0 iff substrate funciona end-to-end.
4. **Producción imagen Docker:** `make docker-build` produce `iris:dev` multi-stage (golang:1.26-alpine build → distroless/static:nonroot runtime) sin shell ni package manager en runtime.
5. **OpenAPI 3.1 documentado:** `make api-docs` genera `docs/api.yaml` desde rutas chi (huma o swaggo); CI drift check (`git diff --exit-code docs/api.yaml`) bloquea PR si docs no committed.

Repo plumbing soporte: LICENSE Apache 2.0 + NOTICE + SPDX pre-commit hook + .gitignore + README pointing-to-spec + `.golangci.yml` + `go mod init github.com/mlorentedev/iris`.

## Out of scope

- **Cualquier código de los SDD-034 posteriores.** NATS envelope (SDD-034b), K8s runtime adapter agent-sandbox wrapping (SDD-034c), pi workers (SDD-034d), HTMX console (SDD-034e), E2E test (SDD-034f), kubelab manifests (SDD-034g) — todos cero código en este PR. Solo stubs `internal/{api,scheduler,runtime,protocol,post-actions,console,db}/` con paquetes vacíos para que `go build ./...` compile.
- **Postgres + agent.yaml + audit chain + HITL + agents Combo 8.** v0 DB es SQLite (per ADR-002). Postgres adapter post-MVP. Agent contract YAML schema + HITL gate primitive + audit chain HMAC (DP-1/DP-2/DP-4) son ADR-005-defined pero impl es SDD-034d. Cero implementación de ninguno aquí.
- **OIDC auth + RBAC + vault MCP integration + Hermes supervision runtime (ADR-008).** ADR-002 los declara pero v0 no implementa. Auth provider stub en `internal/api/` solo expone interface `Provider` con impl `noop`. Vault MCP integration es runtime concern (agents leen vault) — fuera del motor bootstrap.

## Risks / open questions

> **All decisiones cerradas 2026-05-21. Tasks freeze-ready.**

1. **🟡 OpenAPI generator: huma vs swaggo** — DECIDED = **huma** (newer ~5k⭐, code-first type-safe handlers + auto OpenAPI 3.1 from Go types, zero-drift match con sqlc Go-preferred filosofía).
2. **🟡 envconfig library** — DECIDED = **sethvargo/go-envconfig** (modern struct-tag + context-aware + maintained 2026; kelseyhightower archived 2024).
3. **🟢 Logging format final** — DECIDED = **JSON** v0 (Loki ingesta ambos pero JSON future-proofs SDD-029 OTel trace correlation IDs).
4. **🟠 CI/CD scope SDD-034a** — DECIDED = **INCLUDE en este PR** (Opción A). `.github/workflows/ci.yml` se construye distribuidamente: Task 1 añade jobs pre-commit + golangci-lint + go mod verify; Task 2 añade go build/vet/test; Task 3 añade sqlc diff + migrations test; Task 4 añade smoke-test job containerized; Task 5 añade docker build + push + drift check.
5. **🟢 Image registry paths** — DECIDED 2026-05-21 = **dual-push ghcr.io + Docker Hub**. Makefile usa `REGISTRIES` make-var (space-separated) con default `ghcr.io/mlorentedev/iris docker.io/mlorentedev/iris`. Each build tags y pushes a ambos. CI pipeline login a ghcr via `GITHUB_TOKEN` (auto) + login a docker.io via `DOCKER_HUB_USERNAME`/`DOCKER_HUB_TOKEN` secrets (Manu añade post-merge en repo settings). v0.3+ swap a self-hosted `registry.kubelab.live` cuando montes `distribution/distribution` en kubelab (esperado SDD-034g sub-task o aparte). v0.5+ migrate a Harbor cuando primer FDE customer pide enterprise features (vuln scan + replication).
6. **🟢 Micro-defaults (8 picks)** — ACCEPTED all defaults 2026-05-21: SPDX hook = `Lucas-C/pre-commit-hooks`; golangci linters = `errcheck/gosimple/govet/ineffassign/staticcheck/unused/gofmt/goimports/revive/gosec/misspell`; test = stdlib + testify; extra make targets = `test/fmt/lint/clean/generate`; compose service names = `nats`/`worker-pi-stub`; SQLite conn string = WAL+FK pragmas; NOTICE = Apache 2.0 standard.

## Acceptance criteria

> 1:1 con `features.json` — pass-state gating per pattern-feature-list-as-primitive aplica.

- [ ] **AC1: Repo plumbing limpio.** `pre-commit run --all-files` exit 0 (SPDX hooks + gitleaks pasan); `golangci-lint run ./...` exit 0; `go mod verify` exit 0. (matches `sdd-034a-f1-repo-plumbing`)
- [ ] **AC2: Motor boots and serves health endpoints.** `go build ./cmd/motor && ./bin/motor &` + `curl -fsS localhost:8080/healthz` returns 200 + `curl -fsS localhost:8080/readyz` returns 200; `kill -TERM %1` produces graceful shutdown within 15s con structured JSON log de drain. (matches `sdd-034a-f2-motor-skeleton`)
- [ ] **AC3: DB migrations apply + sqlc queries work.** `make migrate-up` applies 0001_init.up.sql creando `teams` table; `sqlc generate` produces zero diff committed; `go test ./internal/db/... -count=1` exit 0. (matches `sdd-034a-f3-db-layer`)
- [ ] **AC4: Dev environment + smoke test green.** `make install-tools && make dev` brings up `nats:4223 + worker-pi-stub + motor (air hot-reload)`; `make smoke-test` ejecuta los 5 checks del bootstrap-contract § 2 y exit 0. (matches `sdd-034a-f4-dev-environment`)
- [ ] **AC5: Docker image builds + OpenAPI doc generated zero-drift.** `make docker-build` produces `iris:dev` image multi-stage non-root; `docker run --rm iris:dev /motor --version` outputs version triplet; `make api-docs` regenera `docs/api.yaml` con zero diff vs committed (`git diff --exit-code docs/api.yaml`). (matches `sdd-034a-f5-docker-openapi`)
- [ ] **AC6: Image tagging soporta dual-registry deployment promotion staging→prod.** `make docker-build` produce un tag local `iris:dev` + loop sobre `REGISTRIES` make-var (default `ghcr.io/mlorentedev/iris docker.io/mlorentedev/iris`) creando para cada registry: `<registry>:<git-sha-short>` (immutable identity para GitOps) + `<registry>:0.1.0-dev` (semver pre-release). `make docker-push` empuja a TODOS los registries del var (CI: ghcr login auto via `GITHUB_TOKEN`, docker.io login via `DOCKER_HUB_USERNAME`+`DOCKER_HUB_TOKEN` secrets — graceful skip si secret missing). Binary embebe `Version` + `GitSHA` + `BuildDate` via `-ldflags "-X main.version=... -X main.gitSHA=... -X main.buildDate=..."`. `docker run --rm iris:dev /motor --version` outputs los 3 fields. **Soporta SDD-034g downstream sin invadir su scope.** v0.3+ swap a `registry.kubelab.live/iris` cuando self-hosted live; v0.5+ Harbor enterprise. (matches `sdd-034a-f5-docker-openapi` — extiende, no añade feature nueva)
- [ ] **AC7: CI pipeline verde + drift gates.** `.github/workflows/ci.yml` corre en push/PR todos los jobs (distribuidos por Tasks 1-5): pre-commit + golangci-lint + go mod verify (Task 1) + go build/vet/test (Task 2) + sqlc diff + migrations test (Task 3) + smoke-test containerized (Task 4) + docker build + OpenAPI drift check (Task 5). Branch protection bloquea merge si CI no verde. (matches new feature `sdd-034a-f6-ci-pipeline` — añadir a features.json)

## References

- Vault: `10_projects/iris/11-tasks.md` (backlog entry SDD-034a)
- ADR: [adr-002-orchestrator-architecture](../../docs/adr/adr-002-orchestrator-architecture.md) (canonical architecture — was kubelab/adr-038, renumbered 2026-05-19)
- ADR: [adr-003-license-apache-2](../../docs/adr/adr-003-license-apache-2.md) (license decision — was kubelab/adr-040)
- Positioning: lives in the vault strategy reports (not in the repo) — 2026-05-21
- ADR: [adr-005-architectural-boundary](../../docs/adr/adr-005-architectural-boundary.md) (vault-as-SSOT + CORE/EDGE + Combo 8 + DP-1..5 + EQ-1 + métricas) — 2026-05-21
- OQ-1 spike: ADOPT `kubernetes-sigs/agent-sandbox` as K8s runtime substrate (consumed by SDD-034c, not SDD-034a) — 2026-05-21
- Convention origin: kubelab/10-roadmap § Stream C: Repo Separation
- Bootstrap contract: `./bootstrap-contract.md` (substrate definition — Q1-Q4 filled 2026-05-17)
- Cross-ref deferred discussion: kubelab staging→prod flow → opens at SDD-034g spec init (post-merge this PR). Helm chart + GitOps + per-env secrets + Authelia + TLS solver scoped to SDD-034g, NOT here.
