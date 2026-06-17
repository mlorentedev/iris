---
tags: [spec, verification, templates]
created: "2026-06-16"
---

# Verification - SDD-034d-worker-skeleton

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (test name / observed behavior). Commit hashes filled in at PR time.

- [x] **AC1 protocol roundtrip** -> `TestDispatchDecodesUserMessage` / `TestDispatchRejectsWrongType` / `TestDispatchRejectsEmptyPrompt` (`internal/worker/dispatch_test.go`); decodes via shared `internal/protocol`, no mirror.
- [x] **AC2 pi in a worktree + cleanup (item C)** -> `TestDriverStreamsEventsToAgentEnd` (`driver_test.go`) + `TestWorktreeLifecycle` (`worktree_test.go`) + `TestHandleHappyPath` (`worker_test.go`, asserts worktree removed via end-to-end run against fake-pi).
- [x] **AC3 activity emission + container_validation** -> `TestActivityFor*` (`activity_test.go`), `TestHandleHappyPath` (1 activity_event per tool_execution_* on `team.acme.activity`), `TestEmitStartup` (container_validation on boot).
- [x] **AC4 graceful shutdown (item A)** -> `TestDriverCancelReapsPi` (`shutdown_test.go`: ctx-cancel terminates pi, Run returns, non-zero exit) + `cmd/worker/main.go` `signal.NotifyContext(SIGINT,SIGTERM)` pull loop.
- [x] **AC5 pi failure surfaced (item B)** -> `TestHandleFailurePath` (`worker_test.go`: fake-pi exits 1 -> failure activity_event + error returned, not swallowed).
- [x] **AC6 smoke-test green (#4/#5)** -> `build/worker/Dockerfile` + `worker-pi` (image `iris-worker:dev`) in `compose.dev.yml` + smoke-test #4 retargeted. Verified locally: substrate up, both containers healthy, `[smoke] 5/5 checks passed` (exit 0); worker logged `worker ready` on `team.dev.worker.worker-1`.

## Test status

- Test suite: `go test ./... ` -> all packages PASS; `internal/worker` coverage 74.8% (gap = defensive log branches + the NATS adapter, which is covered by the smoke-test e2e, not unit).
- Hermetic by design: the entire worker logic is tested against a compiled `fake-pi` stand-in (`testdata/fakepi`) — no tokens, no network. `nats.go` is quarantined to `natsbus.go` behind the `Publisher` seam.
- Lint/format: `golangci-lint run ./...` clean (gosec G204 handled via `runGit` chokepoint + `jobIDRE` input validation, single justified nolint); `gofmt`, `go vet`, `go mod verify` clean.
- Manual smoke test: worker image builds (283 MB, version triplet OK, pi 0.79.6 + git present, non-root); `make smoke-test` -> `[smoke] 5/5 checks passed` against the real `worker-pi` container.
- No regressions in existing test suite: yes (motor/api/db/runtime/protocol all still PASS).

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **pi `--mode json` contract grounded from upstream `docs/json.md`** (proposal residual resolved): prompt is a **positional arg** (`pi --mode json "<prompt>"`); stdout is JSONL — a `session` header then `agent_start`/`turn_start`/`message_*`/`tool_execution_*`/`turn_end` terminating in `agent_end`. The `--mode json` flag the proposal pinned does exist (a first WebFetch only saw `rpc.md`, which is rpc-only — corrected against `README.md` + `json.md`).
- **T1 driver — malformed line vs broken stream.** A single unparseable event line is skipped with a WARN (truncated to 256 B), not fatal: the work product lives in the worktree, not the parser, so one bad telemetry line must not abort a healthy session. A broken *stream* (`scanner.Err()`: dead pipe, line > 1 MiB) IS surfaced as an error so the dispatch layer can raise a failure activity_event (AC5). Distinguishing the two avoids both silent-failure and over-aborting.
- **T1 driver — drain to EOF, never break at `agent_end`.** `agent_end` is recorded as a flag (`SawAgentEnd`), not a stop condition; reading stdout to EOF is what lets `cmd.Wait()` return instead of deadlocking on an unread pipe, and gives a clean child reap on SIGTERM (T4).
- **Exit code is data, not assumed 0.** `Result.ExitCode` is reported via `exitCodeFrom` (uses Go 1.26 `errors.AsType`); the worker never assumes pi exits 0. **Residual:** pi's *actual* exit code on a clean `agent_end` is still to be confirmed against the real binary in T5 (unit tests run hermetically against fake-pi, no tokens/network) — the contract already tolerates either outcome.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons.md`? <yes / no - one line of what>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes / no - one line of what>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes / no - one line>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/SDD-034d-worker-skeleton/` -> `specs/archive/SDD-034d-worker-skeleton/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
