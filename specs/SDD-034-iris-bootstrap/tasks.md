---
tags: [spec, tasks, iris, bootstrap]
created: "2026-05-17"
---

# Tasks - SDD-034-iris-bootstrap

> TDD order. One task = one focused commit. Reorder freely while spec is in `draft` state; freeze once `implementing` starts.

## Setup

- [ ] Branch created from main: `feat/SDD-034-iris-bootstrap`
- [ ] `bootstrap-contract.md` is complete (Q1-Q4 filled, no `[AGENT-DRAFT]` tags)
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

> Populated from `bootstrap-contract.md` § 4 (task breakdown), accepted 2026-05-17. Each task maps to one `features.json` entry — pass-state gating per pattern-feature-list-as-primitive means harness (not agent) transitions `state: passing`.

- [ ] **Task 1: Repo plumbing** (`features.json` id: `sdd-034a-f1-repo-plumbing`) — LICENSE Apache 2.0 + NOTICE + SPDX pre-commit hook + `.gitignore` + README pointing to spec + `go mod init github.com/mlorentedev/iris` + `.golangci.yml`. Verification: `pre-commit run --all-files && golangci-lint version && go mod verify`.

- [ ] **Task 2: Motor skeleton** (`features.json` id: `sdd-034a-f2-motor-skeleton`) — `internal/{api,scheduler,runtime,protocol,post-actions,console,db}/` stubs + `cmd/motor/main.go` with chi router + `/healthz` + `/readyz` + envconfig + slog logfmt + SIGTERM graceful shutdown. Verification: `go build ./... && go vet ./... && ./bin/motor & sleep 1; curl -fsS localhost:8080/healthz && curl -fsS localhost:8080/readyz; kill %1`.

- [ ] **Task 3: DB layer** (`features.json` id: `sdd-034a-f3-db-layer`) — `sqlc.yaml` + `migrations/0001_init.{up,down}.sql` minimal `teams` table + `queries/teams.sql` + sqlc generate + golang-migrate on motor startup. Verification: `make migrate-up && sqlc generate && go test ./internal/db/... -count=1`.

- [ ] **Task 4: Dev environment** (`features.json` id: `sdd-034a-f4-dev-environment`) — Makefile (`dev/install-tools/smoke-test/docker-build/api-docs`) + `.air.toml` + `compose.dev.yml` (`nats:4223` + `worker-pi` stub) + smoke-test script implementing 5 checks. Verification: `make install-tools && make dev & sleep 5; make smoke-test; kill %1`.

- [ ] **Task 5: Production Dockerfile + OpenAPI + version embedding** (`features.json` id: `sdd-034a-f5-docker-openapi`) — multi-stage Dockerfile (golang:1.26-alpine build → distroless/static:nonroot) + `make docker-build` produces **3 simultaneous tags** (`iris:dev` + `iris:<git-sha-short>` + `iris:0.1.0-dev`) + binary embeds `Version`/`GitSHA`/`BuildDate` via `-ldflags "-X main.version=... -X main.gitSHA=... -X main.buildDate=..."` + **huma** integration generating `docs/api.yaml` (decided 2026-05-21 over swaggo per type-safe handlers preference) + CI drift check. Verification: `make docker-build && docker run --rm iris:dev /motor --version | grep -qE '^iris 0\.1\.0-dev \(sha=[a-f0-9]{7,} built=[0-9-]{10}T' && make api-docs && git diff --exit-code docs/api.yaml`. **Soporta SDD-034g downstream sin invadir su scope** (Helm/GitOps/secrets viven en SDD-034g, no aquí).

**Ordering rationale:** 1→2 (SPDX hooks before any real code), 2→3 (motor boots without DB), 3→4 (`make dev` needs migrations on startup), 4→5 (Dockerfile needs binary + make targets exist). 5 closes K8s-readiness — SDD-034g consumes Dockerfile + OpenAPI as its deploy inputs.

**CI/CD distribution (`sdd-034a-f6-ci-pipeline`):** cada Task añade su own jobs al `.github/workflows/ci.yml` en mismo commit que el código.
- Task 1: bootstrap workflow file + jobs `pre-commit` + `golangci-lint` + `go mod verify`
- Task 2: añadir jobs `go build` + `go vet` + `go test`
- Task 3: añadir jobs `sqlc-diff` + `migrations-test`
- Task 4: añadir job `smoke-test` (corre `make smoke-test` en runner)
- Task 5: añadir jobs `docker-build` (no push v0 — `REGISTRY=iris.local` default) + `openapi-drift-check`

**Micro-defaults locked (2026-05-21):** SPDX hook `Lucas-C/pre-commit-hooks`; golangci linters `errcheck/gosimple/govet/ineffassign/staticcheck/unused/gofmt/goimports/revive/gosec/misspell`; test stdlib + testify; extra make targets `test/fmt/lint/clean/generate`; compose service names `nats`/`worker-pi-stub`; SQLite WAL+FK pragmas; NOTICE Apache 2.0 standard.

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command (per pattern-feature-list-as-primitive)
- [ ] Bootstrap-contract acceptance signal verified: peer + clean machine + this doc → passing smoke test
- [ ] Type checks pass (`go vet`, `golangci-lint`)
- [ ] Lint passes
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

See sibling `features.json` (per pattern-feature-list-as-primitive). Pass-state gating applies: the agent CANNOT write `"state": "passing"` — only the harness may.
