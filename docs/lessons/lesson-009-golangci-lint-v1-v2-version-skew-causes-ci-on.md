---
id: lesson-009-golangci-lint-v1-v2-version-skew-causes-ci-on
type: lesson
status: active
created: "2026-06-19"
owner: manu
tags: [iris, lesson]
---

# golangci-lint v1/v2 version skew causes CI-only lint failures

- **Context:** SDD-034d — `make install-tools` does not pin the golangci-lint version. Local environment had v1.62.2; CI (`.github/workflows/ci.yml`) installs v2.12.2.
- **Problem:** gosec checks that v2 flags (e.g. G204 on `exec.CommandContext` with variable args) are silently skipped by v1. Code that passes lint locally can fail CI without any local signal. The skew is invisible unless you check `golangci-lint --version` explicitly.
- **Solution:** install the CI-matching version locally with `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2` before pushing. Long-term: pin the version in `make install-tools` or use the golangci-lint GitHub Action's `version:` pin so local and CI always match.
- **Tags:** golangci-lint, ci, version-skew, gosec, tooling, sdd-034d
