---
id: lesson-006-test-an-agent-supervisor-worker-with-a-fake-s
type: lesson
status: active
created: "2026-06-17"
owner: manu
tags: [iris, lesson]
---

# Test an agent-supervisor worker with a fake subprocess + a publisher seam, not a mock harness

- **Context:** SDD-034d worker drives `pi` over a stdout JSONL stream and emits telemetry on NATS. The risky parts are process spawning, pipe draining, exit codes, and the event→envelope mapping — not the transport.
- **Problem:** mocking at the `os/exec` boundary tests a fiction (the contract is a real subprocess + byte stream), and importing a real NATS broker into unit tests is slow and flaky. But the agent loop needs tokens/network if driven by real pi.
- **Solution:** two seams. (1) A compiled **fake-pi** (`testdata/fakepi`, ignored by `go build ./...`) replays canonical pi JSONL fixtures with knobs for exit code (`FAKEPI_EXIT`) and hang-on-SIGTERM (`FAKEPI_HANG`) — so the driver, worktree, activity mapping, failure path, and ctx-cancel reap are all unit-tested with zero tokens/network. (2) A narrow `Publisher` interface quarantines `nats.go` to one file (`natsbus.go`); the whole job lifecycle is tested against a fake bus that captures envelopes, and the real NATS roundtrip is covered once by the smoke-test. Pyramid: hermetic units + one e2e, no testing theatre.
- **Tags:** testing, fake-subprocess, seam, nats, hermetic, worker, sdd-034d
