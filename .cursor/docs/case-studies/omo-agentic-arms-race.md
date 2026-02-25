# Case Study: Oh My OpenCode — The Agentic Arms Race

**Date:** 2026-02-25  
**Subject:** Oh My OpenCode (OmO) — multi-model agent orchestration harness  
**Source:** `github.com/code-yeongyu/oh-my-opencode` (dev branch, `docs/guide/overview.md`)  
**Purpose:** Competitive analysis. Map OmO concepts to Origami primitives. Identify gaps and advantages. Extract actionable improvements.

---

## 1. What OmO Is

Oh My OpenCode is a multi-model agent orchestration harness built on top of OpenCode (a Cursor/Claude Code alternative). It transforms a single AI agent into a coordinated team of named agents that delegate work to each other based on task type.

**Architecture:** Imperative delegation chain. A main orchestrator (Sisyphus) classifies user intent, then delegates to specialized agents (planner, executor, consultant, search). Each agent is a prompt template with routing rules. A category system abstracts model selection — tasks are tagged as `visual-engineering`, `ultrabrain`, `quick`, or `deep`, and each category maps to a model.

**Named agents:**

| Agent | Role | Origami mental model |
|-------|------|---------------------|
| Sisyphus | Main orchestrator, never stops | Walker traversing the graph |
| Prometheus | Strategic planner (interview mode) | No equivalent — planning persona |
| Atlas | Todo executor, distributes to subagents | WalkTeam + AffinityScheduler |
| Oracle | Read-only architecture consultant | Shadow persona in Dialectic |
| Metis | Gap analyzer — catches what planner missed | Adversarial Dialectic (antithesis) |
| Momus | Ruthless reviewer | Shadow persona (Challenger) |
| Librarian | Documentation and code search | Knowledge source (KnowledgeSourceRouter) |
| Explore | Fast codebase grep | Not an agent concern — tooling |
| Hephaestus | GPT-native autonomous agent | Walker with different element |

**Key features:**
- Intent Gate: classifies request type before routing
- Category system: abstracts model selection (4 categories)
- Provider fallback chains: if provider A is down, try B
- Parallel execution: 5+ background agents simultaneously
- Accumulated wisdom: context from task N informs task N+1
- Discipline enforcement: todo enforcer, comment checker, Ralph Loop
- Hash-anchored edits: `LINE#ID` content hashing for edit reliability
- Skills with embedded MCPs: scoped MCP servers per task

---

## 2. Concept Mapping: OmO to Origami

| OmO Concept | Origami Equivalent | Comparison |
|-------------|-------------------|------------|
| **Named agents** (Sisyphus, Atlas, etc.) | `Persona` (Herald, Seeker, Sentinel, Weaver) with `StepAffinity`, `PromptPreamble`, `PersonalityTags` | Origami is **more formal** — quantified traits via Element dimensions, empirically profiled by Ouroboros. OmO agents are prompt templates with manual routing. |
| **Intent Gate** (classify before routing) | No direct equivalent | **Gap.** Origami pipelines start at a fixed entry node. An "entry classifier" node pattern could serve this purpose declaratively. |
| **Category system** (visual, ultrabrain, quick, deep) | `Element` system (Fire, Water, Earth, Air, Diamond, Lightning) | Origami is **richer** — 6 elements with quantified behavioral traits (`SpeedClass`, `MaxLoops`, `ConvergenceThreshold`, `ShortcutAffinity`, `EvidenceDepth`, `FailureMode`) vs 4 ad-hoc categories. Origami is **empirically grounded** (Ouroboros measures); OmO assigns manually. |
| **Model matching** (category → model) | `ProviderRouter` + `DeriveStepAffinity` + `Ouroboros.ElementMatch` | Origami has the **infrastructure** but the loop is **not closed** — Ouroboros measures, DeriveStepAffinity suggests, but nothing auto-routes at walk time. |
| **Parallel execution** (5+ background agents) | `WalkTeam` + `ParallelEdge` (fan-out via errgroup) | Origami is **more structured** (graph-based parallelism with fan-out/fan-in merge nodes) but **less ad-hoc** (can't fire arbitrary background agents outside the graph). |
| **Accumulated wisdom** (cross-task learning) | `WalkerState` (partial) | **Gap.** Origami has per-walk state but no formal mechanism to carry learnings across separate pipeline walks. |
| **Gap analysis** (Metis) | `Adversarial Dialectic` (D0-D4, Shadow pipeline: indict → discover → defend → hearing → verdict) | Origami is **architecturally deeper** — thesis/antithesis/synthesis with formal Shadow personas, quantified confidence, structured verdict. OmO's Metis is a single-pass review prompt. |
| **Discipline enforcement** (Todo enforcer, Ralph Loop) | `Element.MaxLoops`, `ConvergenceThreshold`, `FailureMode` | Origami has the **traits** but no dedicated **enforcement runtime**. Element traits constrain walk behavior; OmO has explicit loop-and-check agents. |
| **Provider fallback chains** | No equivalent | **Gap.** `ProviderRouter` is static routing with no fallback. If the primary provider fails, the dispatch fails. |
| **Skills with embedded MCPs** | `PipelineServer` with domain hooks, schema-validated `submit_step` | Origami is **more formal** (schema factory, step validation) but MCPs are not scoped per-task — one server per pipeline. |

---

## 3. Competitive Advantages (Origami has, OmO lacks)

### 3.1 Declarative Pipeline DSL

Origami defines pipelines as YAML with typed nodes, conditional edges (`when:` expressions), loops, shortcuts, parallel branches, and a `DONE` pseudo-node. OmO's "pipeline" is imperative delegation — Sisyphus calls Atlas which calls subagents. There is no replayable, inspectable, diffable pipeline artifact.

**Why it matters:** Declarative pipelines are reviewable, testable, and reproducible. You can diff two pipeline versions. You can replay a pipeline with different data. You can visualize it (Kami). Imperative delegation chains are opaque.

### 3.2 Ouroboros Meta-Calibration

Origami empirically profiles models on 6 behavioral dimensions (Speed, Persistence, ConvergenceThreshold, ShortcutAffinity, EvidenceDepth, FailureMode). OmO assigns models to categories manually via config files.

**Why it matters:** Manual assignment doesn't scale. When a new model releases, OmO users must manually test and reconfigure. Ouroboros re-profiles automatically. The Seed Pipeline redesign (dichotomous probing with AI-generated questions) makes this even more robust.

### 3.3 Adversarial Dialectic

Origami has a formal thesis/antithesis/synthesis quality validation pattern with Shadow personas (Challenger, Abyss, Bulwark, Specter). OmO has Momus (a "ruthless reviewer" prompt) and Metis (a "gap analyzer" prompt). These are single-pass reviews, not multi-round dialectics with structured confidence scoring.

**Why it matters:** Single-pass review catches surface issues. Adversarial dialectic forces the system to defend its conclusions against opposition, producing calibrated confidence and identifying genuine uncertainty.

### 3.4 Masks (Detachable Behavioral Middleware)

Origami's Mask system attaches/detaches behavioral modifications to walkers at runtime. No OmO equivalent — agents have fixed prompts.

**Why it matters:** Masks enable runtime behavioral tuning without changing the underlying persona. A "verbose" mask for debugging, a "concise" mask for production, a "skeptical" mask for security review — all composable.

### 3.5 Zones and Stickiness

Origami partitions pipeline nodes into zones with configurable stickiness (0-3) and work-stealing. OmO has no spatial concept — agents are assigned by category, not by pipeline region.

**Why it matters:** Zones enable context locality. A walker that enters the "Investigation" zone stays there (stickiness) until the work is done, accumulating context. Work-stealing lets idle walkers pick up tasks from other zones when needed.

### 3.6 Calibration System (M1-M20)

Asterisk (on Origami) has a scientific calibration system with 20 metrics, ground truth datasets, stub/dry/wet progression, and aggregate reporting. OmO has no calibration — no ground truth, no metrics, no measurement of agent quality.

**Why it matters:** Without calibration, you can't measure improvement. OmO users rely on vibes. Origami users have M19 scores.

### 3.7 WalkObserver + SignalBus (Unified Observability)

Origami has two structured event systems (walk events + coordination signals) being unified into a single stream (Kami EventBridge). OmO mentions no observability infrastructure.

**Why it matters:** You can't debug what you can't see. Kami will provide live visualization, breakpoints, replay, and AI-controlled inspection. OmO agents are black boxes.

### 3.8 Kami Live Debugger

Origami's upcoming Kami debugger (triple-homed: MCP + HTTP/SSE + WS) provides live pipeline visualization, debug control (pause/resume/breakpoints), session recording, and replay. OmO has no equivalent.

**Why it matters:** This is the presentation differentiator. Seeing agents traverse a graph in real-time, with cooperation dialogs and evidence accumulation, is something prompt-template orchestration cannot produce.

---

## 4. Competitive Gaps (OmO has, Origami should learn)

### Gap 1: The routing loop is not closed

OmO's category system is simplistic (4 manual categories) but it **works at runtime**. When a task comes in, OmO routes it to the right model immediately. Origami has all the pieces — Ouroboros profiles models, ElementMatch scores element affinity, DeriveStepAffinity maps steps to preferred traits, ProviderRouter routes to dispatchers — but they're not wired end-to-end. The PersonaSheet (from `ouroboros-seed-pipeline` Phase 7) is the missing connector.

**Actionable:** Wire PersonaSheet into ProviderRouter via an `AutoRouteOption` RunOption. At walk start, load the PersonaSheet for the current model(s), derive provider hints per step, and configure the router accordingly. This is OmO's category system done empirically instead of manually.

### Gap 2: No provider fallback chains

OmO configures fallback chains per provider — if Anthropic is down, try OpenAI, then Cursor. Origami's ProviderRouter is static: one route per provider, no fallback. A dispatch failure is a pipeline failure.

**Actionable:** Add `Fallbacks map[string][]string` to ProviderRouter. On dispatch error from the primary provider, iterate fallbacks in order. Emit `EventProviderFallback` for observability. Small change, high resilience.

### Gap 3: No entry classifier pattern

OmO's Intent Gate classifies user requests before routing — research, implementation, investigation, fix. Origami pipelines start at a fixed entry node. There is no documented pattern for "classify first, then branch."

**Actionable:** This is not a code gap — Origami's DSL already supports it. A classifier node can set `vars.intent`, and downstream edges can use `when: vars.intent == "investigation"`. The gap is documentation. Create a `testdata/patterns/intent-classifier.yaml` example and document the pattern.

---

## 5. Architectural Class Analysis

OmO and Origami operate at fundamentally different architectural levels.

**OmO is prompt engineering + glue code.** Named agents are system prompts. The category system is a JSON config mapping. Provider fallback is a retry loop. Parallel execution is "fire a subagent." There is no formal graph, no typed artifacts, no schema validation, no calibration, no observability. The value is in the prompt craft and the pragmatic wiring.

**Origami is a formal graph-based framework.** Pipelines are declarative YAML. Nodes and edges are typed Go interfaces. Artifacts have schemas. Walkers have quantified behavioral traits. Meta-calibration empirically profiles models. Adversarial dialectic validates conclusions. Masks compose behavior. Zones partition context. Observers stream events. The value is in the architecture — it enables things prompt templates structurally cannot.

**The gap that matters:** OmO ships the "right model for the right task" experience today, despite crude infrastructure, because they close the loop. Origami has superior infrastructure but hasn't closed the loop. The arms race is won by whichever side closes their gaps first.

**Origami's structural advantage:** OmO cannot add calibration, dialectic, graph-based orchestration, or live debugging by writing more prompts. These require architectural investment they haven't made. Origami can close the routing loop with a small wiring change (PersonaSheet → ProviderRouter).

---

## 6. Actionable Takeaways

1. **Close the routing loop** — Wire PersonaSheet into ProviderRouter. This is the single highest-leverage improvement. It transforms Origami from "has all the pieces" to "auto-routes models to tasks empirically." OmO does this manually with 4 categories. Origami would do it with 6 quantified dimensions and empirical profiling. Strictly superior.

2. **Add provider fallback** — Small change, high resilience. ProviderRouter gains `Fallbacks` field and retry logic. Observable via EventProviderFallback. Prevents cascading failure when a provider has an outage.

3. **Document the entry classifier pattern** — Origami's DSL already supports Intent Gate-style classification. A YAML example + documentation makes this discoverable. No code change needed.

4. **Presentation narrative** — The case study itself is a presentation asset. When showing Origami, the competitive framing "here's what the prompt-template approach looks like, here's what the formal-framework approach enables" is compelling. Use it.

---

## References

- OmO overview: `github.com/code-yeongyu/oh-my-opencode/blob/dev/docs/guide/overview.md`
- OmO agent-model matching: `github.com/code-yeongyu/oh-my-opencode/blob/dev/docs/guide/agent-model-matching.md`
- Origami Element system: `element.go` (DefaultTraits, 6 elements with quantified traits)
- Origami Persona system: `persona.go` (8 personas, StepAffinity, PromptPreamble)
- Origami ProviderRouter: `dispatch/provider.go` (Routes map, static routing)
- Origami Ouroboros: `ouroboros/suggest.go` (ElementMatch, DeriveStepAffinity)
- Origami AffinityScheduler: `scheduler.go` (Select by StepAffinity + Element)
- Related contracts: `ouroboros-seed-pipeline` (Phase 7: PersonaSheet), `kami-live-debugger` (observability advantage), `consumer-ergonomics` (API polish)
