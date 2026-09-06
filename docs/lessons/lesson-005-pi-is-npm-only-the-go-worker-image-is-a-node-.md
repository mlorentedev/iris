---
id: lesson-005-pi-is-npm-only-the-go-worker-image-is-a-node-
type: lesson
status: active
created: "2026-06-17"
owner: manu
tags: [iris, lesson]
---

# pi is npm-only — the Go worker image is a Node base, not distroless

- **Context:** SDD-034d picks Go for the fleet worker (ADR-009), partly for the motor's "static distroless binary" deploy story. The worker drives `pi` (earendil-works/pi) as a subprocess.
- **Problem:** pi ships **only via npm** (`npm i -g --ignore-scripts @earendil-works/pi-coding-agent`) — there is **no standalone/compiled binary** (no bun/deno-compile/pkg). So a `distroless/static:nonroot` worker image cannot run pi: it has no Node. The "static binary" advantage cited for Go does **not** carry to the worker's runtime image.
- **Solution:** the worker image is multi-stage Go-build → **Node base** with pi installed `--ignore-scripts` and **version-pinned**, plus the Go binary copied in. pi upgrades are dependency bumps (pin + test). This Node requirement is identical for any worker language, so it does not change the Go-vs-Python decision — it only means the worker image differs from the motor's distroless image. Recorded in ADR-009; relevant to SDD-034g (deploy).
- **Tags:** pi, npm, docker-image, distroless, node, worker, sdd-034d
