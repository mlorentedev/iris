# iris — Documentation

Build/operate documentation for iris, versioned with the code (docs-as-code).
Decide/position and research artifacts (competitive analysis, brainstorm sessions,
project context) live in the maintainer's knowledge vault, not here — per the
knowledge-placement model (build/operate → repo, decide/position → store).

## Architecture Decision Records

| ADR | Status | Decision |
|---|---|---|
| [ADR-001](adr/adr-001-frontend-htmx-go.md) | accepted | Console frontend: HTMX + Go templates |
| [ADR-002](adr/adr-002-orchestrator-architecture.md) | accepted | Architecture (v0 canonical): Go motor + NATS bus + workers |
| [ADR-003](adr/adr-003-license-apache-2.md) | accepted | License: Apache 2.0 |
| [ADR-005](adr/adr-005-architectural-boundary.md) | accepted | Architectural boundary contract (CORE/EDGE + vault-as-SSOT) |
| [ADR-007](adr/adr-007-iris-v0-coding-loop.md) | active | Execute iris v0: pi runtime + NATS + verification gate |
| [ADR-009](adr/adr-009-worker-runtime-go.md) | accepted | Fleet worker is Go (drives pi CLI), not Python — supersedes ADR-002 §Why Python |

> Positioning, pivot and product-strategy reports live in the vault
> (`25-prestudy/reports/`), not in this repo — per the knowledge-placement model above.

> Per-feature specs live in [`specs/`](../specs/). Task tracking lives on the
> bitácora GitHub Project (issues in this repo).
