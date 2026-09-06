---
id: iris-lessons-index
type: index
status: active
created: "2026-08-17"
owner: manu
tags: [iris, lessons, index]
---

# Lessons Learned Index

| # | Date | Title | File | Tags |
|---|---|---|---|---|
| 001 | 2026-06-16 | Docker agent containers must run with an init (PID 1 signal handling) | [[docs/lessons/lesson-001-docker-agent-containers-must-run-with-an-init\|lesson-001-docker-agent-containers-must-run-with-an-init.md]] | docker, runtime, signals, pid1, graceful-shutdown |
| 002 | 2026-06-16 | `ImagePull` always contacts the registry — gate on local presence | [[docs/lessons/lesson-002-imagepull-always-contacts-the-registry-gate-o\|lesson-002-imagepull-always-contacts-the-registry-gate-o.md]] | docker, image-pull, performance, offline, flaky-tests |
| 003 | 2026-06-16 | moby `github.com/docker/docker` is `+incompatible` — pin `go-connections` | [[docs/lessons/lesson-003-moby-github-com-docker-docker-is-incompatible\|lesson-003-moby-github-com-docker-docker-is-incompatible.md]] | go-modules, docker-sdk, plus-incompatible, dependency-pinning, build-break |
| 004 | 2026-06-16 | agent-sandbox releases lag `main` on the API version — pin to `@main`, verify from source | [[docs/lessons/lesson-004-agent-sandbox-releases-lag-main-on-the-api-ve\|lesson-004-agent-sandbox-releases-lag-main-on-the-api-ve.md]] | go-modules, agent-sandbox, kubernetes, api-versioning, pre-1.0, dependency-pinning |
| 005 | 2026-06-17 | pi is npm-only — the Go worker image is a Node base, not distroless | [[docs/lessons/lesson-005-pi-is-npm-only-the-go-worker-image-is-a-node-\|lesson-005-pi-is-npm-only-the-go-worker-image-is-a-node-.md]] | pi, npm, docker-image, distroless, node, worker, sdd-034d |
| 006 | 2026-06-17 | Test an agent-supervisor worker with a fake subprocess + a publisher seam, not a mock harness | [[docs/lessons/lesson-006-test-an-agent-supervisor-worker-with-a-fake-s\|lesson-006-test-an-agent-supervisor-worker-with-a-fake-s.md]] | testing, fake-subprocess, seam, nats, hermetic, worker, sdd-034d |
| 007 | 2026-06-19 | `go get` must run in the foreground before `go mod tidy` | [[docs/lessons/lesson-007-go-get-must-run-in-the-foreground-before-go-m\|lesson-007-go-get-must-run-in-the-foreground-before-go-m.md]] | go-modules, gotcha, backgrounded-tasks, nats, sdd-034d |
| 008 | 2026-06-19 | gosec G204 on subprocess calls — validate input, funnel through one helper, annotate once | [[docs/lessons/lesson-008-gosec-g204-on-subprocess-calls-validate-input\|lesson-008-gosec-g204-on-subprocess-calls-validate-input.md]] | gosec, golangci-lint, G204, subprocess, security, go, sdd-034d |
| 009 | 2026-06-19 | golangci-lint v1/v2 version skew causes CI-only lint failures | [[docs/lessons/lesson-009-golangci-lint-v1-v2-version-skew-causes-ci-on\|lesson-009-golangci-lint-v1-v2-version-skew-causes-ci-on.md]] | golangci-lint, ci, version-skew, gosec, tooling, sdd-034d |
