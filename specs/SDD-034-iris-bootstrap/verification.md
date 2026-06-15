---
tags: [spec, verification, iris, bootstrap]
created: "2026-05-17"
---

# Verification - SDD-034-iris-bootstrap

## Evidence

Every acceptance criterion from `proposal.md` mapped to concrete proof. Tasks
landed as one PR each.

- [x] **AC1: Repo plumbing** -> PR #16. CI jobs `pre-commit` + `golangci-lint` + `go-mod-verify` green.
- [x] **AC2: Motor boots + health endpoints** -> PR #18. `TestHealthEndpoints`; live `/healthz` + `/readyz` 200, SIGTERM graceful drain.
- [x] **AC3: DB migrations + sqlc** -> PR #20. `internal/db` tests (migrate idempotency, teams roundtrip, blank-id CHECK); CI `sqlc-diff` + `migrations-test`.
- [x] **AC4: Dev env + smoke test** -> PR #21. `make smoke-test` -> `[smoke] 5/5 checks passed`; CI `smoke-test` job green.
- [x] **AC5: Docker image + OpenAPI zero-drift** -> this PR. `docker run --rm iris:dev /motor --version` -> version triplet; `make api-docs-diff` clean; CI `docker-build` + `openapi-drift-check`.
- [x] **AC6: Dual-registry tagging** -> this PR. `make docker-build` tags `iris:dev` + `<registry>:{sha,0.1.0-dev}` for ghcr.io + docker.io; `make docker-push` skips registries without secrets.
- [x] **AC7: CI pipeline + drift gates** -> distributed across Tasks 1-5; all jobs run on push/PR.

## Test status

- Test suite: `go test -race -cover ./...` -> pass (`internal/api` 94.7%, `internal/db` 52.5%).
- Manual smoke test: `make smoke-test` (per bootstrap-contract.md section 2) -> `[smoke] 5/5 checks passed`.
- No regressions in existing test suite: yes.

## Decisions made during implementation

- **sqlc kept out of `go.mod`** (pinned release binary, not a `go tool` directive) so its dependency tree never leaks into the runtime module. CI provisions it via `sqlc-dev/setup-sqlc`.
- **Explicit column lists, never `SELECT *`** in sqlc queries -> stable generated code (also dodges a sqlc star-expansion offset bug).
- **`.sql` must be ASCII-only** -> sqlc silently corrupts SQL constants around multi-byte characters (exit 0, broken output). Closed with a pre-commit `sql-ascii-only` guard.
- **`/readyz` ping -> 200/503**, body `{"db":"ok"|"down"}`. The `nats` key is deferred to SDD-034b; smoke check 2 asserts `db:ok` only until then.
- **huma `$schema` transformer disabled** so operational probes keep their exact frozen bodies; OpenAPI Info.Version is a fixed constant so `make api-docs` is deterministic.
- **Dockerfile uses `CMD` not `ENTRYPOINT`** so `docker run iris:dev /motor <subcmd>` passes the full argv (matches the verification form and the K8s deploy).

## Promotion candidates

- [x] Lesson for `iris/90-lessons.md`: **sqlc + multi-byte characters in `.sql` = silent codegen corruption** (exit 0, broken SQL constants). Detection: ASCII-only pre-commit guard. Worth crystallizing.
- [ ] ADR-worthy decision: none new (all within ADR-002/007 envelope).
- [ ] New pattern candidate: none.

## Archive checklist

> Deferred to the deliberate `/spec archive` step once all task PRs are merged.

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/SDD-034-iris-bootstrap/` -> `specs/archive/SDD-034-iris-bootstrap/`
- [ ] Backlog entry in the bitacora board ticked with PR links
- [ ] Promotions above executed (if any)
- [ ] `features.json` all `state: passing` with non-empty `evidence` (harness-gated; agent cannot flip)
