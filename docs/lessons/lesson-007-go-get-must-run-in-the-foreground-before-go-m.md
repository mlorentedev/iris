---
id: lesson-007-go-get-must-run-in-the-foreground-before-go-m
type: lesson
status: active
created: "2026-05-01"
owner: manu
tags: [iris, lesson]
---

# `go get` must run in the foreground before `go mod tidy`

- **Context:** SDD-034d — adding `github.com/nats-io/nats.go` as the first NATS client dependency.
- **Problem:** `go get nats.go@latest` was launched as a background task. The subsequent `go mod tidy` ran while the background write was either in-flight or not yet flushed, stripping the dependency. The next build failed with "no required module provides package github.com/nats-io/nats.go" despite the backgrounded get reporting success.
- **Solution:** always run `go get <pkg>` in the foreground, verify it appears in `go.mod`, *then* run `go mod tidy`. Never background Go module mutation commands.
- **Tags:** go-modules, gotcha, backgrounded-tasks, nats, sdd-034d
