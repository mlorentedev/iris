---
tags: [spec, tasks]
created: "2026-06-16"
---

# Tasks - SDD-034d-worker-skeleton

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `feat/SDD-034d-worker-skeleton`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (pi `--mode json` contract pinned from `docs/json.md`; exit-code-after-`agent_end` is the one residual, confirmed empirically in T1)

## Implementation

> TDD order, one commit each. AC tags refer to `proposal.md` "Acceptance criteria".
> pi `--mode json` contract (grounded against pi `docs/json.md`): prompt is a
> **positional arg** (`pi --mode json "<prompt>"`); stdout is JSONL — a `session`
> header line, then `agent_start` → `turn_start` → `message_*` → `tool_execution_*`
> → `turn_end` → `agent_end` (terminal). The **fake-pi** harness replays these
> exact lines so the whole worker is testable in CI with no tokens / no network.

### T1 — pi driver + fake-pi harness  ·  [AC2 partial]

- [ ] Add `github.com/nats-io/nats.go` to `go.mod` (first NATS client in the repo) — `go mod tidy`, `go mod verify`
- [ ] Write `internal/worker/testdata/fakepi/main.go`: a tiny stand-in for `pi` that replays a fixture of canonical JSONL events to stdout and exits with a configurable code (`FAKEPI_FIXTURE`, `FAKEPI_EXIT`), so tests pin pi's contract without inference
- [ ] Add fixtures `internal/worker/testdata/*.jsonl` (happy path with a `tool_execution_*` pair; failure path)
- [ ] **Write failing test** `driver_test.go`: `Driver.Run(ctx, prompt)` spawns the fake-pi, streams events to a handler callback, stops at `agent_end`, returns the observed exit code
- [ ] Implement `event.go` (decode a pi JSONL line into a typed event by its `type`) + `driver.go` (spawn `pi --mode json <prompt>` with `cmd.Dir`, stream stdout line-by-line, dispatch to handler, `Wait()` for exit)
- [ ] **Empirically confirm** pi's exit code on `agent_end` (record finding in `verification.md` "Decisions"); fold the result into the driver's success/failure contract
- [ ] Refactor: scanner buffer sized for large tool results; no goroutine leak on early ctx cancel

### T2 — protocol roundtrip + dispatch + git worktree  ·  [AC1, AC2 item C]

- [ ] **Write failing test** `dispatch_test.go`: a `user_message` envelope (built via `internal/protocol`) decodes to its `UserMessagePayload` and dispatches the worker — no Pydantic mirror, shared package
- [ ] Implement `dispatch.go`: `protocol.Decode` → assert `TypeUserMessage` → `Unmarshal(&UserMessagePayload)` → derive prompt
- [ ] **Write failing test** `worktree_test.go`: a job creates a fresh `git worktree`, runs the driver there (`cwd` = worktree), and the worktree is **removed on completion** — assert no leaked worktree dir / `git worktree list` is clean
- [ ] Implement `worktree.go` (create per job under a temp base, `git worktree add`; `git worktree remove --force` + prune on completion, even on error)
- [ ] Refactor: worktree path derived from `MessageID`; cleanup is `defer`red so it runs on every exit path

### T3 — activity emission + container_validation  ·  [AC3]

- [ ] **Write failing test** `activity_test.go`: each pi `tool_execution_*` event produces exactly one `activity_event` envelope on `team.<team>.activity`; a test subscriber asserts the envelopes decode via `internal/protocol`
- [ ] Implement `activity.go`: map a pi event → `protocol.ActivityEventPayload` (`Event`, `Target`, `Summary`, opaque `Data`) and publish via `protocol.SubjectFor(team, SubjectActivity)`
- [ ] **Write failing test**: on startup the worker emits a `container_validation` envelope (`ContainerValidationPayload`)
- [ ] Implement startup validation (pi present on PATH, worktree base writable) → `container_validation`
- [ ] **Surface any FROZEN gap**: if the pi→`ActivityEventPayload` mapping cannot be expressed with the current `payloads.go` fields, STOP and record it (needs an ADR per the proposal); do not edit the FROZEN surface unilaterally

### T4 — failure path + graceful shutdown  ·  [AC4, AC5]

- [ ] **Write failing test** `failure_test.go`: fake-pi exits non-zero → the worker emits a **failure** `activity_event` (not swallowed); assert the failure envelope is published
- [ ] Implement the failure branch in the dispatch loop (non-zero exit / stream error → failure `activity_event`, worktree still cleaned)
- [ ] **Write failing test** `shutdown_test.go`: SIGTERM mid-job terminates the in-flight pi child and drains within a bounded timeout — assert the pi child is reaped (no orphan), within `IRIS_SHUTDOWN_TIMEOUT`
- [ ] Implement graceful shutdown in `cmd/worker/main.go`: `signal.NotifyContext(SIGINT, SIGTERM)`; cancel propagates to the pi subprocess (`exec.CommandContext`); bounded drain; `slog` structured logs (mirror the motor's `run()` shape)

### T5 — worker image + compose + smoke-test + CI  ·  [AC6]

- [ ] **Write failing**: extend `scripts/smoke-test.sh` check #4 to assert the real `worker-pi` service (rename from `worker-pi-stub`), and confirm check #5 (NATS roundtrip) still passes
- [ ] Add `build/worker/Dockerfile`: multi-stage Go-build → **Node base** with pi pinned (`--ignore-scripts`) + the Go worker binary copied in (per ADR-009 — NOT distroless)
- [ ] Add `make worker-build` (+ `IMAGE_WORKER` var); replace `worker-pi-stub` in `compose.dev.yml` with the real `worker-pi` (env: `NATS_URL`, `TEAM`, `WORKER_NAME`, inference passthrough), `depends_on: nats healthy`
- [ ] Wire CI: worker in `go build ./... && go test`, a `worker-docker-build` job, and the smoke-test job against the real worker
- [ ] **Doc-reconciliation sweep** (per ADR-009 consequences): `internal/protocol/messages.go` package doc, `adr-001-frontend-htmx-go` §5, `adr-003-license-apache-2` — "Python workers" → "Go / fleet workers"

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] `go build ./...`, `go vet ./...`, `golangci-lint run ./...` pass
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder; iris#5 linked

## Machine-readable features

The sibling `features.json` is the harness-facing contract (per [[pattern-feature-list-as-primitive]]): each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence`.

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit 0, may set that terminal state. Reviewers must reject PRs where `features.json` contains `passing` entries with empty `evidence`.
