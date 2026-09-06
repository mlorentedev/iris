---
id: lesson-004-agent-sandbox-releases-lag-main-on-the-api-ve
type: lesson
status: active
created: "2026-06-16"
owner: manu
tags: [iris, lesson]
---

# agent-sandbox releases lag `main` on the API version — pin to `@main`, verify from source

- **Context:** SDD-034c Kubernetes adapter depends on `sigs.k8s.io/agent-sandbox` for the `Sandbox` CRD Go types.
- **Problem:** `go get sigs.k8s.io/agent-sandbox@latest` resolved the published release **v0.4.6**, but `go mod tidy` then failed with `module ... found (v0.4.6), but does not contain package sigs.k8s.io/agent-sandbox/api/v1beta1`. The released tag still ships the **old `v1alpha1`** API; `main` had already migrated the types to **`v1beta1`** with no intermediate release. The upstream docs (and context7) also still described `v1alpha1` — only the cloned source showed `api/v1beta1` with group `agents.x-k8s.io`.
- **Solution:** pin to `sigs.k8s.io/agent-sandbox@main` (pseudo-version `v0.4.7-0.<ts>-<sha>`) so the `v1beta1` types resolve. agent-sandbox is pre-1.0 and fast-moving: verify the API group/version from the actual cloned source, not the docs, and re-pin to a real tag once one ships `v1beta1`. The entire dependency is isolated to `internal/runtime/k8s` (ADR-005 / OQ-1), so the re-pin is a one-package change.
- **Tags:** go-modules, agent-sandbox, kubernetes, api-versioning, pre-1.0, dependency-pinning
