# Contract — visual-editor

**Status:** draft  
**Goal:** Ship a Visual Editor for Origami pipelines — drag-and-drop graph builder, bidirectional YAML sync, run management dashboard — with Community (open source) and Enterprise (commercial) editions following the Ansible open-core model.  
**Serves:** Polishing & Presentation (vision)

## Contract rules

Global rules only, plus:

- **Separate product.** The Visual Editor is NOT part of the Origami framework. It consumes Origami's structured output (WalkObserver events, Artifact data, Mermaid diagrams, PipelineDef). The framework has no dependency on the editor.
- **Red border respected.** "No web UI" remains true for the framework itself. This is a separate product built on the framework's output.
- **Two editions, one codebase.** Community and Enterprise editions share the same codebase. Enterprise features are gated by license, not by separate repositories.
- **AWX model for Community Edition.** The Community Edition is fully functional for single-user use — not a crippled trial. Enterprise features (RBAC, multi-tenancy, audit, SSO) only activate with an enterprise license.
- **Kami integration, not replacement.** The Visual Editor embeds Kami's graph visualization and event streaming. Kami remains the developer debugger; the Visual Editor is the operational management plane.
- **Red Hat brand compliance.** All UI colors must use the RH color system defined in `docs/rh-presentation-dna.md`. Element-to-color mapping per Section 2. Enterprise Edition must use [PatternFly](https://www.patternfly.org/) (Red Hat's open-source design system) for all UI components. Community Edition: PatternFly recommended, RH color collections required.

## Context

- `strategy/origami-vision.mdc` — Product Topology: "Future UI product — pipeline definitions, run history, artifact inspection, visualization."
- `contracts/completed/framework/origami-pipeline-studio.md` — Completed design-only contract. Architecture sketch, API contract, data model. This contract supersedes Pipeline Studio with implementation scope.
- `contracts/draft/kami-live-debugger.md` — EventBridge, KamiServer (triple-homed), Debug API, React frontend. The Visual Editor shares Kami's React Flow graph component and EventBridge data source.
- `contracts/draft/origami-lsp.md` — Language Server for pipeline YAML. The Visual Editor embeds Monaco + LSP for the YAML editing pane.
- `contracts/draft/origami-collections.md` — Collection format, FQCN resolution. The Visual Editor's component palette shows available collections and their contents.
- `docs/case-studies/visual-editor-landscape.md` — Case study of Excalidraw, Mermaid, and Ansible Automation Controller. Business model analysis recommending the Ansible open-core model.
- `docs/case-studies/ansible-collections.md` — Ansible Collections and Automation Hub as second revenue stream (certified content).
- `docs/rh-presentation-dna.md` — Red Hat brand color system, element-to-color mapping, accessibility constraints. All Visual Editor UI must comply.

### Current architecture

```mermaid
flowchart LR
    subgraph today [Today - CLI only]
        CLI["origami run\norigami validate"]
        Render["Render -> Mermaid"]
        Observer["WalkObserver\nLogObserver / TraceCollector"]
    end

    Dev["Developer"] --> CLI
    CLI --> Observer
    CLI --> Render
```

No visual interface. Pipeline authoring is YAML-only. Run monitoring is terminal output. Artifact inspection requires CLI queries or log parsing. No operational management layer for teams.

### Desired architecture

```mermaid
flowchart TB
    subgraph origami_fw [Origami Framework]
        Walk["Walk / WalkTeam"]
        Observer["WalkObserver"]
        Render["Render PipelineDef"]
        Colls["Collection Registry"]
        LSPSrv["Language Server"]
    end

    subgraph kami_pkg [Kami Package]
        EB["EventBridge"]
        Debug["Debug API"]
    end

    subgraph ve_app [Visual Editor]
        subgraph fe [Frontend - React]
            GraphUI["Pipeline Graph\nReact Flow"]
            YAMLPane["YAML Editor\nMonaco + LSP"]
            RunDash["Run Dashboard"]
            ArtView["Artifact Inspector"]
            Palette["Component Palette\nfrom Collections"]
        end
        subgraph be [Backend - Go]
            API["REST/GraphQL API"]
            Store["Event Store\nPostgres / SQLite"]
            StudioObs["StudioObserver\nWalkObserver adapter"]
            Scheduler["Run Scheduler"]
        end
        subgraph enterprise [Enterprise Features]
            RBAC["RBAC Engine"]
            Audit["Audit Trail"]
            SSO["SSO Gateway"]
            Mesh["Mesh Coordinator"]
        end
    end

    Observer --> EB
    EB --> StudioObs
    StudioObs --> API
    Render --> GraphUI
    Colls --> Palette
    LSPSrv --> YAMLPane
    Debug --> GraphUI
    API --> Store
    Store --> fe
```

## FSC artifacts

| Artifact | Target | Compartment |
|----------|--------|-------------|
| Visual Editor product spec | `docs/visual-editor-product.md` | domain |
| StudioObserver adapter design | `docs/studio-observer.md` | domain |
| Business model decision record | `docs/case-studies/visual-editor-landscape.md` | domain |
| Edition feature matrix | `docs/visual-editor-editions.md` | domain |

## Execution strategy

This is the largest product scope in Origami's roadmap. Execution is split into phases that each deliver a usable increment. Phase 0 establishes Playwright E2E testing and Kami integration from day 1 — every subsequent phase is gated by green smoke tests. Phase 1 delivers a read-only viewer (graph + run history). Phase 2 adds the pipeline builder (drag-and-drop + bidirectional YAML). Phase 2.5 adds subgraph fold/unfold. Phase 2.7 adds diagnostic overlays (heatmaps, traces, diffs). Phase 3 adds run management (launch, schedule, monitor). Phase 3.5 adds the Agentic Editor (AI-powered graph modification). Phase 4 adds enterprise features (RBAC, multi-tenancy, audit, SSO, collaboration cursors, alert rules). Phase 5 adds the collections palette and certified content integration. Phase 5.5 adds product polish (themes, analytics, export, templates, onboarding).

Dependencies are strict: Kami (EventBridge + React Flow) must ship before Phase 1. LSP must ship before Phase 2. Collections Phase 3.5 (SubgraphNode) must ship before Phase 2.5, but Phase 2.5 can be developed in parallel using mock nested `PipelineDef` data. Collections must ship before Phase 5. Playwright E2E (Phase 0) gates every subsequent phase — no PR merges without green E2E smoke tests.

## Feature tiers

Every feature maps to one of three audience tiers. Tiers are cumulative — Tier 2 includes all of Tier 1, Tier 3 includes all of Tier 1+2.

- **Tier 1: PoC Must-Have** — Proves the Visual Editor works. Internal team validation. "The graph renders, edits sync, runs animate." Minimum viable product.
- **Tier 2: QE Division Demo Must-Have** — Impresses QE leadership. Agent visibility, run diagnostics, pipeline optimization. "This changes how we do test failure analysis."
- **Tier 3: CEO Matt Hicks Must-Have** — Product-level. Business model viability, enterprise readiness, multi-domain applicability. "This is the next Ansible Automation Controller."

### Tier 1 — PoC Must-Have

| Feature | Phase | Task |
|---------|-------|------|
| Playwright E2E from day 1 (`window.__origami` bridge) | 0 | PW1-PW6 |
| Kami integration for live debugging | 0 | PW5-PW6 |
| Pipeline graph visualization | 1 | V5 |
| YAML editor with LSP | 2 | B1 |
| Bidirectional YAML-graph sync | 2 | B2 |
| Live graph animation (node enter/exit) | 3 | R2 |
| Run history dashboard | 1 | V6 |
| Artifact inspector | 1 | V7 |
| Auto-layout (dagre/ELK) | 2 | B10 |
| Dark mode (RH color system dark variants) | 2 | B11 |
| Keyboard-first navigation (Tab/Arrow/Enter/Esc) | 2 | B12 |
| Command palette (Ctrl+K) | 2 | B13 |

### Tier 2 — QE Division Demo Must-Have

| Feature | Phase | Task |
|---------|-------|------|
| Subgraph fold/unfold | 2.5 | SV1-SV5 |
| Run comparison (side-by-side) | 3 | R3 |
| Run replay | 3 | R4 |
| Drag-and-drop node palette | 2 | B3 |
| Edge drawing + condition builder | 2 | B4 |
| Zone editor | 2 | B5 |
| Walker editor | 2 | B6 |
| Heatmap overlay (latency, cost, errors) | 2.7 | DO1 |
| Walker trace overlay (per-walker path) | 2.7 | DO2 |
| Pipeline diff view (visual git diff) | 2.7 | DO3 |
| Persona cards (element + radar chart) | 2.7 | DO4 |
| Dialectic visualizer (thesis/antithesis/synthesis) | 2.7 | DO5 |
| Node health indicator (green/yellow/red badge) | 2.7 | DO6 |
| Cost estimator (pre-run token estimate) | 2.7 | DO7 |
| Pipeline testing mode (dry-run with stubs) | 2.7 | DO8 |
| Edge path explorer (highlight all paths) | 2.7 | DO9 |
| Semantic zoom (detail by zoom level) | 2.7 | DO10 |
| Lasso select + bulk operations | 2.7 | DO11 |
| Contextual node inspector (rich side panel) | 2.7 | DO12 |
| Split view (graph + any panel) | 2.7 | DO13 |

### Tier 3 — CEO Matt Hicks Must-Have

| Feature | Phase | Task |
|---------|-------|------|
| Agentic Editor mode (AI builds/modifies graph from intent) | 3.5 | AE1-AE3 |
| RBAC engine | 4 | E1 |
| Audit trail | 4 | E2 |
| SSO gateway | 4 | E3 |
| Multi-tenancy | 4 | E4 |
| Centralized logging | 4 | E5 |
| Topology viewer | 4 | E7 |
| Collaboration cursors (Enterprise real-time) | 4 | E8 |
| Alert rules (PagerDuty, Slack, webhooks) | 4 | E9 |
| Collections palette | 5 | C1-C5 |
| Theme system (domain-specific visual identity) | 5.5 | PP1 |
| Pipeline analytics dashboard (trends, P95, cost) | 5.5 | PP2 |
| Affinity matrix (walker-to-node fit) | 5.5 | PP3 |
| Export to presentation (SVG/PNG/Mermaid) | 5.5 | PP4 |
| Embeddable graph component (npm package) | 5.5 | PP5 |
| Node templates (personal + team library) | 5.5 | PP6 |
| Annotations / sticky notes on graph | 5.5 | PP7 |
| Undo/redo visual timeline | 5.5 | PP8 |
| Onboarding tour (guided first-use) | 5.5 | PP9 |
| Touch/tablet support (iPad demo) | 5.5 | PP10 |

## Coverage matrix

| Layer | Applies | Rationale |
|-------|---------|-----------|
| **Unit** | yes | API handlers, StudioObserver event mapping, RBAC permission checks, audit event recording |
| **Integration** | yes | Full backend startup (API + EventBridge), SSE streaming to frontend, YAML-to-graph bidirectional sync |
| **Contract** | yes | API schema stability (REST/GraphQL), StudioObserver event format, RBAC permission model |
| **E2E** | yes | Playwright from day 1 (Phase 0). Smoke tests: graph renders, YAML round-trip, node drag. Integration: Kami SSE streaming, agentic workflow MCP loop. `window.__origami` bridge for all assertions. |
| **Concurrency** | yes | Multiple SSE clients, concurrent run management, multi-tenant isolation |
| **Security** | yes | Web application security (OWASP full checklist), RBAC enforcement, audit completeness, SSO integration |

## Tasks

### Phase 0 — Playwright E2E + Kami integration (day 1)

Testing infrastructure that gates every subsequent phase. Adopts the Demiurge pattern from Hegemony (`/home/dpopsuev/Projects/hegemony`): `window.__origami` bridge for Playwright access, graceful service skip for optional backends, factory separation for test isolation. Every PR must pass Phase 0 smoke tests before merge.

- [ ] **PW1** Playwright config — `playwright.config.ts` with `testDir: ./e2e`, Chromium, `webServer` auto-starts Vite dev server. GPU launch flags for React Flow canvas rendering (`--use-gl=angle`, `--enable-gpu-rasterization`, `--ignore-gpu-blocklist`) — same pattern as Hegemony.
- [ ] **PW2** `window.__origami` bridge — expose runtime API on `window` for Playwright: `snapshot()` (full graph state), `nodeCount()`, `edgeCount()`, `selectedNode()`, `zoomLevel()`, `foldState()`, `yamlContent()`. Mirrors Hegemony's `window.__perf` pattern. TypeScript types for all return shapes.
- [ ] **PW3** Graceful service skip — `requireKami(page)` helper that checks WS on Kami port and `test.skip`s when unreachable. Tests split into standalone (graph render, YAML sync — no backend) and integration (live run, agentic editor — Kami required).
- [ ] **PW4** Smoke tests — graph renders with correct node count, YAML round-trip preserves structure, node drag updates position, edge creation works. These tests gate every PR from day 1.
- [ ] **PW5** Kami integration test fixture — connect to Kami's WS server (EventBridge), verify live node state updates push to `window.__origami`, agent position dots animate on node enter/exit events.
- [ ] **PW6** Agentic workflow E2E — AI sends a command via Kami's MCP tools (e.g. `highlight_nodes`, `zoom_to_zone`), Playwright verifies the graph updates in real-time via `window.__origami`. Tests the full loop: MCP -> Kami -> EventBridge WS -> React Flow -> `window.__origami`.

### Phase 1 — Read-only viewer (depends on: Kami)

- [ ] **V1** Implement `StudioObserver` — `WalkObserver` adapter that sends KamiEvents to the Visual Editor API
- [ ] **V2** Implement Event Store schema (runs, events, artifacts) with SQLite for Community and Postgres for Enterprise
- [ ] **V3** REST API: `GET /pipelines`, `GET /runs`, `GET /runs/:id/events` (SSE stream), `GET /runs/:id/artifacts/:node`
- [ ] **V4** React scaffold: Vite + TypeScript + Tailwind + React Flow. Configure Tailwind theme with RH Color Collection 1 tokens per `docs/rh-presentation-dna.md`. Enterprise Edition: use PatternFly components.
- [ ] **V5** Pipeline graph component — render PipelineDef as interactive React Flow graph with zone backgrounds, element colors, and node status indicators
- [ ] **V6** Run history dashboard — list past runs with status, duration, pipeline name, walker count
- [ ] **V7** Artifact inspector — click a node in a completed run to see its input/output artifacts
- [ ] **V8** `go:embed frontend/dist/*` — single binary with embedded SPA
- [ ] **V9** `origami studio --port 8080` CLI command
- [ ] **V10** Integration test: start StudioObserver, walk a pipeline, verify events appear in Event Store, verify SSE stream delivers to frontend

### Phase 2 — Pipeline builder (depends on: LSP)

- [ ] **B1** YAML editor pane — Monaco editor connected to Origami LSP via WebSocket
- [ ] **B2** Bidirectional sync engine: graph changes generate YAML diffs; YAML changes update graph model
- [ ] **B3** Drag-and-drop node palette — built-in node families (generic, transformer types) plus registered transformers
- [ ] **B4** Edge drawing — click source node, click target node, configure `when:` condition via expression builder
- [ ] **B5** Zone editor — create/resize/recolor zones, assign nodes, configure stickiness
- [ ] **B6** Walker editor — define WalkerDefs (name, element, persona, preamble, step affinity) via form UI
- [ ] **B7** Pipeline validation — call `origami validate` on every change, show diagnostics inline on graph and in YAML pane
- [ ] **B8** Export — download pipeline as `.yaml` file, copy Mermaid diagram to clipboard
- [ ] **B9** Unit tests: bidirectional sync (graph -> YAML -> graph roundtrip preserves structure)
- [ ] **B10** Auto-layout — integrate dagre (fast) and ELK (high-quality) layout engines. "Auto-arrange" button in toolbar. Layout respects zone boundaries — nodes stay within their assigned zone. User can lock node positions to prevent auto-layout from moving them.
- [ ] **B11** Dark mode — RH color system dark variants for all components. Automatic detection via `prefers-color-scheme`. Manual toggle in toolbar. All RH Color Collection 1 tokens have dark-mode counterparts. Contrast ratios meet WCAG AA.
- [ ] **B12** Keyboard-first navigation — Tab cycles through nodes, Arrow keys move between connected nodes, Enter opens inspector, Esc closes panels. Focus ring visible on active node. All mouse interactions have keyboard equivalents.
- [ ] **B13** Command palette — Ctrl+K opens fuzzy search over: nodes (by name/type), edges, zones, walkers, recent runs, actions (add node, export, validate). Same UX as VS Code command palette.

### Phase 2.5 — Subgraph fold/unfold (depends on: Collections Phase 3.5)

Subgraph visualization with IDE-like collapse/expand. Can be developed in parallel with Collections using mock nested `PipelineDef` data. Applies to both the read-only viewer (Phase 1) and the pipeline builder (Phase 2) — placed here because the graph model extension is needed before run management.

- [ ] **SV1** Hierarchical graph model — extend the React Flow graph data model to represent `SubgraphNode`s as collapsible group nodes. When collapsed, show as a single node with a fold indicator (chevron or depth badge, like IDE code folding). When expanded, reveal the subgraph's internal nodes/edges inline, visually nested inside a container with a subtle border and indented background.
- [ ] **SV2** Fold/unfold interaction — click the fold indicator to toggle. Keyboard shortcut (Ctrl+Shift+[ / Ctrl+Shift+]) for fold/unfold, matching IDE conventions. "Fold All" / "Unfold All" toolbar buttons. Depth-aware: folding a parent also folds all children. Unfolding only reveals one level at a time (like IDE indent folding).
- [ ] **SV3** Edge routing across levels — edges that cross subgraph boundaries render as dashed lines entering/exiting the collapsed node. When expanded, edges connect to the actual internal source/target nodes. Animated transition on fold/unfold so the user doesn't lose spatial context.
- [ ] **SV4** Breadcrumb navigation — when zoomed deep into a nested subgraph (2+ levels), show a breadcrumb trail at the top: `Root > Investigation > Correlation`. Clicking a breadcrumb folds everything below that level and zooms to fit.
- [ ] **SV5** Minimap depth — the React Flow minimap shows collapsed subgraphs as single rectangles, expanded ones as grouped rectangles. Matches the current fold state.

### Phase 2.7 — Diagnostic overlays

Overlays that turn the graph from a static diagram into an operational dashboard. Each overlay is independently toggleable. Data sources: completed run artifacts, live SSE events, and `PipelineDef` metadata.

- [ ] **DO1** Heatmap overlay — color nodes by latency (green-yellow-red gradient), token cost (blue gradient), or error rate (orange gradient). Toggle between metrics in overlay toolbar. Legend with scale.
- [ ] **DO2** Walker trace overlay — select a walker from a completed run, highlight the exact path it took through the graph. Multiple traces can be shown simultaneously with distinct colors. Dim non-traversed nodes.
- [ ] **DO3** Pipeline diff view — select two pipeline YAML versions (from git or run history), render both graphs side-by-side with added/removed/changed nodes highlighted (green/red/yellow). Like a visual `git diff` for pipelines.
- [ ] **DO4** Persona cards — click an agent in a run to see its persona: element affinity radar chart (6 axes), personality traits, model profile, step affinity heatmap. Uses RH element colors from `docs/rh-presentation-dna.md`.
- [ ] **DO5** Dialectic visualizer — for nodes using Adversarial Dialectic, show thesis/antithesis/synthesis flow as a three-column panel. Animate the progression. Show confidence scores at each stage.
- [ ] **DO6** Node health indicator — badge on each node showing aggregate health: green (>95% success rate), yellow (80-95%), red (<80%). Based on last N runs. Tooltip shows exact stats.
- [ ] **DO7** Cost estimator — before launching a run, estimate total token cost per node based on historical averages. Show as node badges and a total in the launch dialog. Warn if estimate exceeds configurable threshold.
- [ ] **DO8** Pipeline testing mode — "Dry Run" button that executes the pipeline with stub extractors returning canned data. Validates flow, edge conditions, and zone transitions without LLM calls. Results shown as a test report overlay.
- [ ] **DO9** Edge path explorer — click any node, highlight all reachable paths from that node (forward) or all paths leading to it (backward). Show path probabilities based on edge conditions and historical data.
- [ ] **DO10** Semantic zoom — at low zoom: nodes show only name and health badge. At medium zoom: add type, zone, last run status. At high zoom: show full configuration, recent artifacts, metrics. Transitions are smooth.
- [ ] **DO11** Lasso select + bulk operations — draw a selection rectangle to select multiple nodes. Bulk actions: move to zone, delete, set breakpoint, export subgraph, apply tag. Shift+click to add/remove from selection.
- [ ] **DO12** Contextual node inspector — rich side panel that appears on node click. Tabs: Config (YAML source), Runs (history for this node), Artifacts (latest input/output), Metrics (latency, cost, errors over time), Code (extractor source if available).
- [ ] **DO13** Split view — drag-to-resize split between graph and any panel (YAML, inspector, run history, artifacts). Supports horizontal and vertical splits. Remembers layout per pipeline.

### Phase 3 — Run management

- [ ] **R1** Launch button — select pipeline, configure vars, choose walkers, start run
- [ ] **R2** Live graph animation — nodes light up on enter/exit, edges animate on transition, artifacts appear as badges
- [ ] **R3** Run comparison — side-by-side view of two runs of the same pipeline (diff artifacts, diff timing)
- [ ] **R4** Run replay — load recorded JSONL, play back through the graph visualization (reuse Kami Replayer)
- [ ] **R5** Run scheduling — cron-style schedules for recurring pipeline runs (Enterprise only)
- [ ] **R6** Run notifications — webhook on completion/failure (Enterprise only)

### Phase 3.5 — Agentic Editor

AI-powered graph construction and modification. The user describes intent in natural language; an AI agent autonomously builds or modifies the pipeline graph while the user watches changes happen live. Uses Kami's MCP tools for graph manipulation and EventBridge for real-time feedback.

- [ ] **AE1** Agentic Editor mode — toggle in toolbar switches to "AI Assistant" mode. Chat input at bottom of graph. AI agent (connected via Kami MCP) can: add/remove nodes, create/delete edges, configure zone assignments, set edge conditions, modify walker definitions. Each AI action is animated on the graph in real-time via EventBridge WS.
- [ ] **AE2** Intent-to-graph pipeline — user types natural language (e.g. "Add a retry loop around the triage node with max 3 attempts"), the AI decomposes into graph operations, executes them sequentially through Kami MCP tools, and the graph updates live. Undo button reverts the entire AI action sequence as one unit.
- [ ] **AE3** Optimization suggestions — when heatmap data is available (Phase 2.7 DO1), the AI can analyze bottlenecks and suggest graph modifications: "Node X has P95 latency of 12s — suggest adding a cache node before it" or "Edge Y is never taken — consider removing it." User accepts/rejects each suggestion.

### Phase 4 — Enterprise features

- [ ] **E1** RBAC engine — roles (admin, operator, viewer), teams, org-scoped permissions on pipelines, runs, collections
- [ ] **E2** Audit trail — every action (create pipeline, launch run, modify RBAC, install collection) logged with actor, timestamp, diff
- [ ] **E3** SSO gateway — LDAP, SAML 2.0, OIDC integration for enterprise authentication
- [ ] **E4** Multi-tenancy — organization isolation (separate namespaces, credentials, pipelines, run history)
- [ ] **E5** Centralized logging — aggregated run logs with search, filtering, and export
- [ ] **E6** License gate — enterprise features activate only with valid license key; Community runs fully without one
- [ ] **E7** Topology viewer — visualize execution topology (workers, zones, providers, mesh nodes)
- [ ] **E8** Collaboration cursors — real-time multi-user presence on the graph. Each user gets a colored cursor (based on their persona element color). See who is editing which node. Requires WebSocket presence channel. Enterprise only.
- [ ] **E9** Alert rules — configure alerts on pipeline events: run failure, latency threshold exceeded, cost budget exceeded, node error rate spike. Delivery: PagerDuty, Slack webhook, email, generic webhook. Enterprise only.

### Phase 5 — Collections integration (depends on: Collections)

- [ ] **C1** Component palette integration — browse installed collections, show available transformers, extractors, nodes, pipelines
- [ ] **C2** Collection installer — `Install` button wraps `origami collection install` with progress feedback
- [ ] **C3** FQCN autocomplete in YAML editor — LSP provides collection-aware completion
- [ ] **C4** Certified badge — visual indicator for enterprise-certified collections (from registry)
- [ ] **C5** Collection dependency viewer — show which collections a pipeline uses, their versions, and update availability

### Phase 5.5 — Product polish

Features that elevate the Visual Editor from a tool to a product. Each is independently valuable; none blocks the others.

- [ ] **PP1** Theme system — domain-specific visual identity beyond RH colors. Consumers register a theme (icon set, node shapes, color overrides, logo) that reskins the entire editor. Default theme uses RH branding. Achilles theme uses security-oriented iconography. Custom themes loadable from Collections.
- [ ] **PP2** Pipeline analytics dashboard — aggregate metrics across runs: P50/P95/P99 latency per node, total token cost trends, error rate over time, walker efficiency (nodes visited vs total). Filterable by date range, pipeline version, walker. Charts via lightweight library (e.g. Recharts).
- [ ] **PP3** Affinity matrix — heatmap showing walker-to-node fit scores based on Ouroboros profiling data. Rows = walkers (personas), columns = nodes, cells = affinity score. Helps users assign the right persona to each pipeline step.
- [ ] **PP4** Export to presentation — one-click export of the current graph view as SVG, PNG, or Mermaid code. Includes zone backgrounds, node labels, edge conditions. SVG is editable in design tools. Mermaid is pasteable into docs. PNG includes the current overlay state (heatmap, traces).
- [ ] **PP5** Embeddable graph component — publish the React Flow pipeline viewer as a standalone npm package. Consumers can embed pipeline visualizations in their own apps (dashboards, docs, Storybook). Minimal dependencies. Configurable theme.
- [ ] **PP6** Node templates — personal and team-scoped template library. Save a node configuration (type, extractor, zone, common settings) as a template. Drag from template palette to create pre-configured nodes. Enterprise: team-shared template library with approval workflow.
- [ ] **PP7** Annotations / sticky notes — place text annotations anywhere on the graph canvas. Markdown-supported. Attach annotations to specific nodes or edges, or float freely. Useful for documenting design decisions, known issues, or review comments.
- [ ] **PP8** Undo/redo visual timeline — sidebar showing a chronological list of all graph mutations. Click any point to jump to that state. Branching: undo, make a different change, and the timeline forks. Persistent across sessions (stored in Event Store).
- [ ] **PP9** Onboarding tour — guided first-use walkthrough. Highlights key UI areas (graph, YAML pane, run dashboard, command palette) with tooltips. Skippable. Restartable from settings. Adapts to edition (Community vs Enterprise).
- [ ] **PP10** Touch/tablet support — responsive layout that works on iPad-sized screens. Touch gestures: pinch-to-zoom, two-finger pan, long-press for context menu, tap-and-hold for drag. Useful for CEO demos on tablets and conference presentations.

### Phase 6 — Validate and tune

- [ ] **T1** Validate (green) — `go build ./...`, `go test ./...`, all E2E tests pass. Visual Editor starts, graph renders, runs stream, YAML syncs.
- [ ] **T2** Tune (blue) — performance (virtualized React Flow for 100+ node pipelines), UX polish (keyboard shortcuts, responsive layout), accessibility (ARIA labels, screen reader support).
- [ ] **T3** Validate (green) — all tests still pass after tuning.

## Acceptance criteria

**Given** a pipeline YAML with 5+ nodes across 2 zones,  
**When** loaded in the Visual Editor,  
**Then** the graph renders with correct topology, zone backgrounds, element colors, and node labels. Clicking a node shows its configuration. The YAML pane shows the source YAML with LSP diagnostics.

**Given** a user drags a new node onto the graph and connects it with an edge,  
**When** the edge `when:` condition is configured,  
**Then** the YAML pane updates in real-time with the new node and edge definition. Saving produces a valid pipeline YAML that passes `origami validate`.

**Given** a pipeline run in progress,  
**When** viewed in the Visual Editor,  
**Then** the graph animates node enter/exit events in real-time via SSE. Completed nodes show artifact badges. Clicking a completed node shows the artifact inspector with input/output data.

**Given** an Enterprise Edition with RBAC configured,  
**When** a user with "viewer" role attempts to launch a run,  
**Then** the launch button is disabled. The audit trail records the denied action. Only users with "operator" or "admin" role can launch runs.

**Given** a pipeline with a SubgraphNode containing 3 internal nodes,  
**When** the subgraph node's fold indicator is clicked to expand,  
**Then** the internal graph unfolds inline with animated transition, edges reconnect to internal nodes, and the breadcrumb shows the subgraph name. Clicking the fold indicator collapses it back to a single node. Keyboard shortcut (Ctrl+Shift+[/]) toggles fold state.

**Given** a Visual Editor dev server running,  
**When** `npx playwright test` executes with no Kami backend,  
**Then** standalone smoke tests pass: graph renders with correct node count via `window.__origami.nodeCount()`, YAML round-trip preserves structure via `window.__origami.yamlContent()`, node drag updates position. Integration tests (Kami, agentic) are skipped with clear "Kami unreachable" message.

**Given** a Visual Editor with Kami running and a pipeline walk in progress,  
**When** Playwright connects and calls `window.__origami.snapshot()`,  
**Then** the snapshot includes live node states (active/completed/pending), agent positions, and the current fold state of all subgraphs. Kami integration tests pass: SSE events stream to `window.__origami`, agent position dots animate.

**Given** the Agentic Editor mode is active,  
**When** the user types "Add a validation node after triage with a confidence threshold of 0.8",  
**Then** the AI agent decomposes the intent into graph operations (add node, add edge, set condition), executes them via Kami MCP tools, and the graph animates each change in real-time. `window.__origami.nodeCount()` increases by 1. Undo reverts all changes as one unit.

**Given** the Community Edition without an enterprise license,  
**When** the Visual Editor starts,  
**Then** all Phase 1-3 features work fully. Enterprise feature menus (RBAC, audit, SSO, mesh) show "Enterprise Edition" badges but do not block any single-user functionality.

## Security assessment

| OWASP | Finding | Mitigation |
|-------|---------|------------|
| A01 Access Control | Web UI exposes pipeline state, run history, and execution controls. | Community: localhost-only by default. Enterprise: RBAC on every API endpoint. `--bind` flag for explicit network exposure. |
| A02 Cryptographic Failures | Run artifacts may contain sensitive data. | Encrypt Event Store at rest (Enterprise). TLS for all API traffic. No secrets in artifact display without credential masking. |
| A03 Injection | YAML editor content displayed in graph, artifact data rendered in inspector. | Sanitize all display content. CSP headers. No `dangerouslySetInnerHTML`. Monaco sandboxed editor. |
| A05 Misconfiguration | Default deployment could expose Visual Editor without auth. | Community: localhost-only, no auth needed. Enterprise: auth required by default, no anonymous access. |
| A07 Authentication | Enterprise SSO integration, session management. | Standard session handling. CSRF tokens. Secure cookies. HttpOnly, SameSite=Strict. |
| A09 Logging & Monitoring | Audit trail completeness. | Enterprise audit logs every API mutation. Structured logging with correlation IDs. Log rotation and retention policies. |

## Notes

2026-02-25 — Contract created from case study `visual-editor-landscape.md`. Business model: Ansible open-core (free Community Edition, paid Enterprise Edition). The Visual Editor is the "money maker" — not because it gates basic functionality, but because it serves enterprise governance, scale, delegation, and visibility needs. Supersedes the design-only `origami-pipeline-studio` contract. Dependencies: Kami (Phase 1), LSP (Phase 2), Collections (Phase 5).

2026-02-25 — Feature tier matrix added. All features mapped to three audience tiers: PoC (Tier 1), QE Division Demo (Tier 2), CEO Matt Hicks Demo (Tier 3). New phases: Phase 0 (Playwright E2E + Kami integration — gates all PRs from day 1), Phase 2.7 (diagnostic overlays), Phase 3.5 (Agentic Editor), Phase 5.5 (product polish). Playwright pattern adopted from Hegemony's Demiurge: `window.__origami` bridge mirrors `window.__perf`, graceful service skip, factory separation, orphan guard. Kami integration provides the testing surface for agentic workflow E2E.
