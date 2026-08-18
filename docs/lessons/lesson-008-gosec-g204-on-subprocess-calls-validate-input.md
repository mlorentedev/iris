---
id: lesson-008-gosec-g204-on-subprocess-calls-validate-input
type: lesson
status: active
created: "2026-05-01"
owner: manu
tags: [iris, lesson]
---

# gosec G204 on subprocess calls — validate input, funnel through one helper, annotate once

- **Context:** SDD-034d `internal/worker/worktree.go` runs `git worktree add/remove` and `git branch -D` with internally-derived paths and branch names. golangci-lint (v2, gosec enabled) flagged every `exec.CommandContext` call as G204 ("subprocess launched with variable").
- **Problem:** blanket `//nolint:gosec` per call site is noise and masks the real security concern (unsanitized external input reaching shell commands). gosec v2 is stricter than v1 and will flag these even when the binary is a literal string, if arguments contain non-literal variables.
- **Solution:** (1) validate the input source with a strict allowlist regex (`jobIDRE ^[A-Za-z0-9][A-Za-z0-9-]{0,127}$`) that rejects path-traversal chars and git ref separators *before* any path or branch string is constructed; (2) funnel all `git` invocations through a single `runGit()` helper (fixed binary, validated argv, no shell); (3) place exactly ONE `//nolint:gosec // G204: fixed binary + validated argv, no shell` on the exec call inside that helper. The fix addresses the underlying concern rather than just silencing the linter.
- **Tags:** gosec, golangci-lint, G204, subprocess, security, go, sdd-034d
