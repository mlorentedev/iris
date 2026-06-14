# iris

> Agent-factory engine: a Go motor + NATS bus + pi fleet workers + a two-stage
> verification gate. Apache-2.0.

iris dispatches coding agents (pi workers) against labelled issues, runs them in
isolated git worktrees, and gates every change — coder self-check + independent QA
agent + human sign-off — before merge.

## Status

**v0 in execution.** Architecture of record:
[ADR-002](docs/adr/adr-002-orchestrator-architecture.md); execution decision:
[ADR-007](docs/adr/adr-007-iris-v0-coding-loop.md).

The active bootstrap spec is
[`specs/SDD-034-iris-bootstrap`](specs/SDD-034-iris-bootstrap/proposal.md) — it
defines the Go motor skeleton, DB layer, dev environment, and production image.

## Documentation

- [`docs/`](docs/) — build/operate docs ([ADRs](docs/adr/)).
- [`specs/`](specs/) — per-feature specs (Spec-Driven Development).

Task tracking lives on the bitácora GitHub Project (issues in this repo).

## License

[Apache-2.0](LICENSE) · see [NOTICE](NOTICE).
