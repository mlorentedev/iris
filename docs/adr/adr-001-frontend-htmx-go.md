---
id: adr-001-frontend-htmx-go
type: adr
status: accepted
created: "2026-05-16"
owner: manu
tags: [iris, orchestrator, frontend, htmx, architecture]
---

# ADR-001 — iris Console Frontend: HTMX + Go templates

> **Status:** Accepted, 2026-05-16
> **Closes:** SDD-027
> **Related:** [adr-003-license-apache-2](./adr-003-license-apache-2.md), [adr-002-orchestrator-architecture](./adr-002-orchestrator-architecture.md), agent-crew-analysis, candidate-team-leader-workers, orchestrator-terminology

## Decision

iris's operator console will be built with **HTMX + Go's `html/template` package**, served from the same Go binary that runs iris's HTTP API. Templates are embedded into the binary via `embed.FS`. No JavaScript build pipeline. No Node.js dependency. Single-binary deploy.

## Scope

What this ADR covers:
- iris's operator console (UI used by clients to configure teams, agents, schedules, webhooks, post-actions, knowledge base; observe activity; manage organization)
- Static assets (CSS via Tailwind compiled at build time via the Go-native `tailwindcss` standalone binary, no Node.js)
- Server-rendered HTML + HTMX-driven partial updates
- Server-Sent Events (SSE) for the realtime activity stream

What this ADR does NOT cover:
- Public marketing site (separate concern, separate stack — likely Astro per consistency with mlorente.dev, deferred until needed)
- Mobile clients (no current plan)
- Embeddable widgets in client products (out of scope; the console is a complete UI, not a library)

## Context

iris targets self-hosted multi-agent platform delivery for freelance + corporate consultoría clients (D18, see orchestrator-terminology). Each client receives an iris deployment they operate themselves. The wedge messaging is "self-hosted, no lock-in, single binary, minimal ops".

The frontend choice must reinforce — not undermine — that wedge. It must also avoid the architectural failure modes observed in the closest prior art (`agent_crew`, see agent-crew-analysis): React + raw `useEffect` + Context with no state library, producing the cluster of "UI flow problems" visible in their open issues (state drift, stale data, double fetches).

Two stacks were considered: **Astro+Islands** (with React/Solid/Svelte for interactive components) and **HTMX + Go templates**.

## Options considered

### Astro+Islands

**Pros:**
- Mature ecosystem (Tailwind, shadcn, React Query, every form/dashboard library imaginable)
- TipTap or other rich text editors integrate cleanly as Islands
- SSR + selective hydration → fast initial loads + interactivity where needed
- Consistency with mlorente.dev (same author already uses Astro)
- Strong talent pool if hiring later

**Cons:**
- Two deployment artifacts (Astro site + Go API) OR awkward serve-Astro-static-from-Go
- Build pipeline requires Node.js + npm + Vite (extra CI/CD step, extra container image)
- Fragments the "single binary" wedge — clients now operate two things or one binary with bundled-but-separate UI
- Requires explicit state-library discipline (React Query, Zustand, or Effect) to avoid `agent_crew`'s drift bugs — discipline that drifts under solo-dev time pressure

### HTMX + Go templates

**Pros:**
- Single binary deploy. `embed.FS` packs all templates and assets into the Go executable. Clients receive ONE artifact: `./iris serve --config=...`
- Zero JavaScript build pipeline. No Node, no npm, no Vite, no Astro CLI. `go build` produces the entire console.
- Whole class of client-state-drift bugs becomes structurally impossible (there is no client state to drift — every interactive update is a server-rendered partial swap).
- Operational simplicity for consultoría delivery: one process, one container image, one binary, one set of logs.
- Server-Sent Events native — perfect fit for the realtime activity stream from candidate-team-leader-workers.
- Tailwind works fine via the standalone Go-native binary (no Node).
- Coherent with backend stack (Go motor + Go templates) — one language across one more layer.

**Cons:**
- Smaller frontend library ecosystem; complex stateful UIs require hand-rolling or careful HTMX extension use.
- Rich text editors at TipTap-class fidelity require vanilla JS integration (TipTap headless works but loses the React DX).
- Smaller talent pool than React.
- HTMX is "boring tech" but not "default tech" — slight ramp-up vs Astro for the current author.
- Limited interactivity ceiling. If the console evolves to drag-drop visual workflow builder (n8n-class), HTMX becomes awkward.

## Why HTMX + Go templates

The decision rests on **wedge alignment + operational simplicity + structural bug avoidance**, in that order:

### 1. Wedge alignment

The first sentence of every consultoría sales conversation is "self-hosted, single binary, no lock-in, deploy in minutes". Astro fragments that sentence. HTMX makes it literally true: `./iris serve` and the entire UI is up. That moment is a sales artifact, not just an ops convenience.

### 2. Operational simplicity for consultoría

Every client deployment Manu supports is a unit of consultoría margin. Operational simplicity scales support directly. ONE thing to deploy, ONE thing to log, ONE thing to upgrade. Astro would force every client deployment to also include a Node-build artifact lifecycle — multiplying support surface.

### 3. Structural bug avoidance

`agent_crew`'s open issues show that React + `useEffect` + Context without a state library produces a predictable class of bugs (stale data, double fetches, race conditions on tab focus). The fix is explicit state-library discipline (React Query, Zustand). That discipline survives in teams with frontend engineers; it tends to drift in solo-dev contexts under time pressure. HTMX bypasses the entire class — there is no client state to drift.

### 4. Performance is a side effect, not the goal

Performance was mentioned in the conversation as the deciding factor; this ADR corrects that framing for the record. HTMX's performance characteristics (server-rendered HTML, minimal payload, no hydration) are excellent but they are not why the choice is correct. The choice is correct because of wedge alignment, ops simplicity, and structural bug avoidance. Tailwind under HTMX produces aesthetically identical UIs to Tailwind under React — the visual outcome is the same. The architectural outcome is what differs.

### 5. Coherent stack — Go motor + Go templates + Go fleet workers

iris's worker is also Go (ADR-009: the worker is a thin pi-supervisor, pi being a TypeScript subprocess dependency), which *strengthens* this argument rather than weakening it: motor, templates, and workers share one language and one toolchain. Adding Go templates for the UI keeps the boundary tight: workers are isolated, frontend is in-process with backend. The maintenance surface is one language for the iris codebase (pi is a pinned external dependency, not source we maintain).

## The two conditions under which we'd switch to Astro

This ADR is reversible. The architecture (Go API + JSON endpoints) supports either frontend. If either of the following becomes true, re-open the decision:

1. **Rich text WYSIWYG editor at TipTap-class fidelity becomes a hard requirement.** If agent instructions evolve into formatted documents with mentions, embedded code blocks with syntax highlighting, tables, drag-drop variables — Astro+Islands+TipTap is materially better than hand-rolled HTMX equivalents. (Today: agent instructions are technical text; Markdown + live preview is sufficient.)
2. **The console grows into a visual workflow builder.** If users expect a node-graph editor for designing agent flows (n8n-class, ComfyUI-class), HTMX cannot meet that ceiling without becoming a worse version of a React+ReactFlow app. (Today: workflows are defined via forms and agent collaboration, not visual graphs.)

Neither condition is on the current roadmap. If either appears, the migration cost is one engineering month (rewrite the affected screens as Astro routes, keep the Go API unchanged).

## n8n coexistence (D19, 2026-05-16)

iris does NOT bundle, depend on, or wrap n8n. It is, however, **HTTP-protocol compatible** with n8n in both directions as a free side effect of existing patterns:

- **n8n → iris (upstream trigger):** iris's webhook trigger system (`POST /webhook/trigger/:token`) accepts inbound calls from any HTTP source, including n8n workflows. Use case: n8n workflow detects a new lead in client's CRM → calls iris webhook → iris dispatches an AI team to act on the lead.
- **iris → n8n (downstream action):** iris's candidate-post-action-bindings (PostAction = arbitrary outbound HTTP) targets any HTTP endpoint, including n8n webhooks. Use case: AI team completes a task → iris fires PostAction → n8n receives the event → n8n workflow handles the 400+ integrations (CRM update, Slack message, email, database write) that iris deliberately does not implement.

This is a clean separation: **n8n handles "boring glue" (400+ integrations); iris handles AI workloads**. Clients who already operate n8n get integration for free; clients who don't get a self-contained iris that works standalone.

**What this ADR explicitly does NOT commit to:**
- Building first-party n8n nodes for iris (could be a side project later; not an MVP deliverable).
- Coupling to n8n's specific API or message format (stay HTTP+JSON generic).
- Bundling n8n in any iris deployment artifact.
- Treating n8n as iris's configuration plane (rejected — defeats single-binary wedge and adds Node + DB dependency + Sustainable Use License complications).

## CSRF protection (mandatory for cookie sessions)

Cookie-based session auth requires CSRF protection because every HTMX request automatically sends cookies. `SameSite=Strict` mitigates cross-site CSRF but does NOT cover all attack vectors (subdomain-hosted attacker pages, certain top-level navigation flows, browser quirks). Implementation requirements:

1. **Per-session CSRF token** issued at login, stored in session, also exposed via a meta tag in the base layout: `<meta name="csrf-token" content="{{.CSRFToken}}">`.
2. **HTMX global `hx-headers`** injects the token on every state-changing request: `<body hx-headers='{"X-CSRF-Token": "{{.CSRFToken}}"}'>`.
3. **Server middleware rejects state-changing requests** (POST, PUT, DELETE, PATCH) missing or mismatching the token. Return 403 with pattern-agent-oriented-errors format.
4. **Token rotation on login + privilege escalation.** Tokens are session-bound; logout invalidates.
5. **Test the CSRF flow** in CI — at least one Playwright test that confirms a request without the header is rejected.

## Implementation checklist

When implementing the console:

1. **Templates under `internal/console/templates/`** organised as `layouts/`, `pages/`, `partials/`. Use Go's `html/template` package.
2. **`embed.FS` at `internal/console/console.go`** packs all templates and static assets into the Go binary. Verified via `go build && file iris` showing single executable.
3. **HTMX vendored as `internal/console/static/htmx-<version>.min.js`** (recommended for offline / air-gapped consultoría deployments). Pin to a specific HTMX version in the filename + a checksums file (`SHA256SUMS`).
4. **Tailwind via standalone Go-native binary** (https://github.com/tailwindlabs/tailwindcss/releases) invoked from `Makefile` at build time. Note: Tailwind 4 is "config-less" in the sense that it does not require `tailwind.config.js` (it uses CSS-first config via `@theme` directive), but it still requires an input CSS file with `@import "tailwindcss"` and a build pass. No `package.json`, no Node.
5. **SSE endpoint for activity stream** at `/api/teams/:id/activity/stream`. Wraps the team's NATS `activity` subject subscription per candidate-team-leader-workers. Use `text/event-stream` content type, send activity events as JSON-serialized lines.
6. **HTMX SSE extension** (`hx-ext="sse"`) on the activity panel template for connecting to `/api/teams/:id/activity/stream`.
7. **Auth as cookie-based session with CSRF** (see CSRF section above). Middleware sets a `Secure HttpOnly SameSite=Strict` cookie after login + CSRF token on every state-changing endpoint.
8. **Forms use HTMX `hx-post` with `hx-target` for partial replacement.** Server returns the updated section's HTML, HTMX swaps it in. No JSON-fetch-then-render dance.
9. **No build step beyond `go build` + `tailwindcss` invocation** in `Makefile`. Verifiable with `make build` on a clean machine with only Go + the tailwindcss binary installed (no Node, no npm).
10. **Document the architecture in `README.md`** with one paragraph + one diagram showing: client browser → HTMX → Go server → templates → embedded assets. No "frontend setup" section, because there is no frontend setup.

## Consequences

**Positive:**
- Single-binary deploy preserved. Wedge messaging stays true.
- Whole class of state-drift bugs (the kind `agent_crew` is fighting) becomes structurally impossible.
- One language across motor + templates + fleet workers (ADR-009). Maintenance surface minimised.
- Build pipeline: `go build` produces the entire iris including UI. CI/CD trivial.
- Offline / air-gapped client deployments work without ceremony (vendor HTMX + tailwindcss binary).

**Negative / constraints:**
- Future migration to Astro (if the two trigger conditions arise) is real work — one engineering month.
- Talent pool for HTMX is smaller. Hiring a frontend engineer down the road has higher friction.
- Complex stateful UIs (multi-step wizards with branching state, drag-drop graph editors) are awkward in HTMX. If those become required, this is the failure point.
- CSRF discipline must be maintained — every state-changing endpoint needs the middleware; missing the middleware is a security hole.

## Re-evaluation triggers

Revisit this decision if:
- A rich text editor at TipTap-class fidelity becomes a hard MVP requirement (condition 1 above).
- The console roadmap evolves to include a visual workflow builder (condition 2 above).
- A consultoría client procurement gate requires a "modern frontend framework" justification (rare, but possible).
- Hiring a frontend engineer is on the immediate roadmap and HTMX is a recruitment blocker.
