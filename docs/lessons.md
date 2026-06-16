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
