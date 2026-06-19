# Lessons

> Build/operate lessons for iris (per the knowledge-placement model: repo-specific
> know-how lives here, cross-project methodology lives in the maintainer's vault).
> Format per lesson: **Context / Problem / Solution / Tags**.

## Docker agent containers must run with an init (PID 1 signal handling)

- **Context:** SDD-034c Docker substrate — each agent is one container; `Stop` calls `ContainerStop` (graceful SIGTERM, then SIGKILL after a timeout).
- **Problem:** the conformance suite's teardown took ~10s per agent. A process running as **PID 1** does not get the kernel's default signal dispositions, so `busybox sleep` (and any agent process) **ignores SIGTERM** — `ContainerStop` waited the full 10s timeout before SIGKILL on every Stop.
- **Solution:** create containers with `HostConfig.Init = true`. Docker injects a tiny init (tini) as PID 1 that forwards signals to the agent and reaps zombies. Teardown became instant and it is the correct posture for agents that spawn child processes anyway.
- **Tags:** docker, runtime, signals, pid1, graceful-shutdown

## `ImagePull` always contacts the registry — gate on local presence

- **Context:** `docker.Runtime.Start` pulled the image before creating the container.
- **Problem:** `client.ImagePull` is **not** a local short-circuit — it always hits the registry to check the manifest. Pulling on every `Start` made the call network-bound and flaky (a stalled pull turned one conformance subtest into a 169s failure).
- **Solution:** `ensureImage` lists local images with a `reference=<ref>` filter and only calls `ImagePull` when none match. `Start` is now fast and offline-tolerant once an image is cached.
- **Tags:** docker, image-pull, performance, offline, flaky-tests

## moby `github.com/docker/docker` is `+incompatible` — pin `go-connections`

- **Context:** adding the Docker SDK (`github.com/docker/docker/client` at `v28.3.3+incompatible`).
- **Problem:** moby ships **no `go.mod`** (hence `+incompatible`), so `go mod tidy` has no constraint on its transitive deps and resolved `github.com/docker/go-connections@v0.7.0`. v0.7.0 **removed `sockets.DialPipe`**, which docker v28.3.3's `client` package still references → `undefined: sockets.DialPipe` build failure. Running `go get go-connections@latest` makes this worse, not better.
- **Solution:** pin `github.com/docker/go-connections v0.5.0` (the last line with `sockets.DialPipe`) in `go.mod`. When bumping the docker SDK, re-verify the compatible `go-connections` version rather than letting `@latest` win.
- **Tags:** go-modules, docker-sdk, plus-incompatible, dependency-pinning, build-break

## agent-sandbox releases lag `main` on the API version — pin to `@main`, verify from source

- **Context:** SDD-034c Kubernetes adapter depends on `sigs.k8s.io/agent-sandbox` for the `Sandbox` CRD Go types.
- **Problem:** `go get sigs.k8s.io/agent-sandbox@latest` resolved the published release **v0.4.6**, but `go mod tidy` then failed with `module ... found (v0.4.6), but does not contain package sigs.k8s.io/agent-sandbox/api/v1beta1`. The released tag still ships the **old `v1alpha1`** API; `main` had already migrated the types to **`v1beta1`** with no intermediate release. The upstream docs (and context7) also still described `v1alpha1` — only the cloned source showed `api/v1beta1` with group `agents.x-k8s.io`.
- **Solution:** pin to `sigs.k8s.io/agent-sandbox@main` (pseudo-version `v0.4.7-0.<ts>-<sha>`) so the `v1beta1` types resolve. agent-sandbox is pre-1.0 and fast-moving: verify the API group/version from the actual cloned source, not the docs, and re-pin to a real tag once one ships `v1beta1`. The entire dependency is isolated to `internal/runtime/k8s` (ADR-005 / OQ-1), so the re-pin is a one-package change.
- **Tags:** go-modules, agent-sandbox, kubernetes, api-versioning, pre-1.0, dependency-pinning

## pi is npm-only — the Go worker image is a Node base, not distroless

- **Context:** SDD-034d picks Go for the fleet worker (ADR-009), partly for the motor's "static distroless binary" deploy story. The worker drives `pi` (earendil-works/pi) as a subprocess.
- **Problem:** pi ships **only via npm** (`npm i -g --ignore-scripts @earendil-works/pi-coding-agent`) — there is **no standalone/compiled binary** (no bun/deno-compile/pkg). So a `distroless/static:nonroot` worker image cannot run pi: it has no Node. The "static binary" advantage cited for Go does **not** carry to the worker's runtime image.
- **Solution:** the worker image is multi-stage Go-build → **Node base** with pi installed `--ignore-scripts` and **version-pinned**, plus the Go binary copied in. pi upgrades are dependency bumps (pin + test). This Node requirement is identical for any worker language, so it does not change the Go-vs-Python decision — it only means the worker image differs from the motor's distroless image. Recorded in ADR-009; relevant to SDD-034g (deploy).
- **Tags:** pi, npm, docker-image, distroless, node, worker, sdd-034d

## Test an agent-supervisor worker with a fake subprocess + a publisher seam, not a mock harness

- **Context:** SDD-034d worker drives `pi` over a stdout JSONL stream and emits telemetry on NATS. The risky parts are process spawning, pipe draining, exit codes, and the event→envelope mapping — not the transport.
- **Problem:** mocking at the `os/exec` boundary tests a fiction (the contract is a real subprocess + byte stream), and importing a real NATS broker into unit tests is slow and flaky. But the agent loop needs tokens/network if driven by real pi.
- **Solution:** two seams. (1) A compiled **fake-pi** (`testdata/fakepi`, ignored by `go build ./...`) replays canonical pi JSONL fixtures with knobs for exit code (`FAKEPI_EXIT`) and hang-on-SIGTERM (`FAKEPI_HANG`) — so the driver, worktree, activity mapping, failure path, and ctx-cancel reap are all unit-tested with zero tokens/network. (2) A narrow `Publisher` interface quarantines `nats.go` to one file (`natsbus.go`); the whole job lifecycle is tested against a fake bus that captures envelopes, and the real NATS roundtrip is covered once by the smoke-test. Pyramid: hermetic units + one e2e, no testing theatre.
- **Tags:** testing, fake-subprocess, seam, nats, hermetic, worker, sdd-034d

## `go get` must run in the foreground before `go mod tidy`

- **Context:** SDD-034d — adding `github.com/nats-io/nats.go` as the first NATS client dependency.
- **Problem:** `go get nats.go@latest` was launched as a background task. The subsequent `go mod tidy` ran while the background write was either in-flight or not yet flushed, stripping the dependency. The next build failed with "no required module provides package github.com/nats-io/nats.go" despite the backgrounded get reporting success.
- **Solution:** always run `go get <pkg>` in the foreground, verify it appears in `go.mod`, *then* run `go mod tidy`. Never background Go module mutation commands.
- **Tags:** go-modules, gotcha, backgrounded-tasks, nats, sdd-034d

## gosec G204 on subprocess calls — validate input, funnel through one helper, annotate once

- **Context:** SDD-034d `internal/worker/worktree.go` runs `git worktree add/remove` and `git branch -D` with internally-derived paths and branch names. golangci-lint (v2, gosec enabled) flagged every `exec.CommandContext` call as G204 ("subprocess launched with variable").
- **Problem:** blanket `//nolint:gosec` per call site is noise and masks the real security concern (unsanitized external input reaching shell commands). gosec v2 is stricter than v1 and will flag these even when the binary is a literal string, if arguments contain non-literal variables.
- **Solution:** (1) validate the input source with a strict allowlist regex (`jobIDRE ^[A-Za-z0-9][A-Za-z0-9-]{0,127}$`) that rejects path-traversal chars and git ref separators *before* any path or branch string is constructed; (2) funnel all `git` invocations through a single `runGit()` helper (fixed binary, validated argv, no shell); (3) place exactly ONE `//nolint:gosec // G204: fixed binary + validated argv, no shell` on the exec call inside that helper. The fix addresses the underlying concern rather than just silencing the linter.
- **Tags:** gosec, golangci-lint, G204, subprocess, security, go, sdd-034d

## golangci-lint v1/v2 version skew causes CI-only lint failures

- **Context:** SDD-034d — `make install-tools` does not pin the golangci-lint version. Local environment had v1.62.2; CI (`.github/workflows/ci.yml`) installs v2.12.2.
- **Problem:** gosec checks that v2 flags (e.g. G204 on `exec.CommandContext` with variable args) are silently skipped by v1. Code that passes lint locally can fail CI without any local signal. The skew is invisible unless you check `golangci-lint --version` explicitly.
- **Solution:** install the CI-matching version locally with `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2` before pushing. Long-term: pin the version in `make install-tools` or use the golangci-lint GitHub Action's `version:` pin so local and CI always match.
- **Tags:** golangci-lint, ci, version-skew, gosec, tooling, sdd-034d
