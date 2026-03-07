# Agentic Hierarchy Design

Design reference for the Origami Operator API: a four-role agentic hierarchy (Broker, Manager, Worker, Enforcer) orchestrated by a Kubernetes Operator-style reconciliation loop, with meta-circuits as the keystone primitive.

Produced during brainstorming session: [Agentic hierarchy and roles](65013565-a183-40d2-ae82-707267f65454), 2026-03-05.

## The Vision

An IDE that treats AI agents as first-class citizens, with a layered hierarchy of specialized roles:

- **Worker Agents** write code. They receive scoped tasks (specific files, specific instructions) and produce diffs. Cheap, parallelizable, disposable.
- **Manager Agents** receive a desired state, analyze the current state, compute the delta, and produce prompts for Worker Agents. They never write code — they produce plans.
- **Enforcer Agents** run system audits: performance, functionality, structure. They are lateral (not in the command chain) and can flag, block, or veto work.
- **Broker Agent** is the human-facing interface. It translates human intent into structured goals, listens to agent reports, and manages human attention.

Origami provides the "rail tracks," "railroad," and "central command" for these agents.

## The Railroad Metaphor

| Railroad Concept | Origami Primitive | Role |
|---|---|---|
| Rail tracks | Edges | Communication channels between agents |
| Stations | Nodes | Work locations where agents operate |
| Train | Walker | Agent moving along tracks, carrying cargo |
| Cargo | Artifacts | Typed data passed between stations |
| Switchyard | Dispatch | Signal routing and coordination |
| Track layout | Circuit (Graph + YAML DSL) | The topology of agent interaction |
| Train identity | Persona + Mask | Who the agent is and what it specializes in |
| Safety inspector | Adversarial Dialectic | Structured challenge of agent output |
| System inspector | Ouroboros | Self-improvement / metacalibration |
| Station procedures | Hooks | Cross-cutting concerns injected at nodes |
| Central Command | Operator (reconciliation loop) | Watches the railroad, dispatches trains, reroutes on failure |

## Role Analysis

### Broker

The only agent that speaks Human. It does not relay; it translates. A human says "the tests are flaky," and the Broker must decide: is this a Worker task (fix a test), a Manager task (investigate flake patterns), or an Enforcer task (audit stability metrics)?

The Broker carries the most context — project history, goals, constraints, conversation memory. There is exactly one per session. Stochastic by nature (translating human intent is inherently ambiguous).

### Manager

Receives desired state + analyzes current state = produces the delta. The Manager never writes code. It produces prompts — scoped, file-specific instructions for Workers. A good Manager minimizes Worker scope ("edit lines 40-60 of `params.go`, replace the concrete struct with the interface") because narrow scope means cheaper Workers, fewer conflicts, and easier rollback.

Managers are few but expensive — they need architectural understanding. Stochastic (decomposition and planning require reasoning).

### Worker

Cheap, parallelizable, disposable. Sees only what the Manager gives it: a file list, a task description, and relevant context. Produces diffs. Has no opinion about whether the task is correct — that is the Manager's job.

Workers are the only agents that write to disk. Predominantly deterministic (apply known patterns) with stochastic fallback for genuinely novel code.

### Enforcer

The independent auditor. Lateral, not in the command chain. Observes outputs from any layer and produces findings that route to the appropriate authority. Three severity levels:

- **Info (flag)** — annotate, no action required
- **Warning (block)** — pause artifact propagation until acknowledged
- **Error (veto)** — abort the sub-circuit; the Manager must re-plan

A functionality Enforcer (tests fail) reports to the Manager. A structural Enforcer (architecture violation) escalates to the Broker. Predominantly deterministic (lint, test, audit all have deterministic answers).

## The D/S Boundary at the Organizational Level

The hierarchy makes the Deterministic/Stochastic boundary explicit at the role level:

| Role | Primary Mode | Why |
|---|---|---|
| Worker | Deterministic with Stochastic fallback | Apply known patterns; LLM only for genuinely novel code |
| Manager | Stochastic | Decomposition and planning require reasoning |
| Enforcer | Deterministic | Lint, test, audit — all have deterministic answers |
| Broker | Stochastic | Translating human intent is inherently ambiguous |

The most expensive tokens (stochastic processing) are concentrated in the fewest agents (one Broker, few Managers). The most numerous agents (Workers, Enforcers) are predominantly deterministic. This is the right cost curve.

## Meta-Circuits: The Central Insight

The most important architectural decision: **a Manager's output is itself a circuit definition.**

Today, a Walker traverses a static graph defined in YAML. But a Manager discovers the work decomposition at runtime — it reads the codebase, computes the delta, and produces a plan. The number of Workers, their file assignments, their parallelism — all emerge from analysis.

The resolution: **nested circuit execution**. A node in an outer circuit can produce a `CircuitDef` as its artifact. The framework walks that generated circuit as a sub-walk, and the sub-walk's output becomes the outer node's artifact.

This preserves every Origami property:

- **Observable**: the generated circuit is YAML — you can inspect exactly what the Manager planned.
- **Calibratable**: you can evaluate the Manager's plan quality independently from the Workers' execution quality.
- **Auditable**: Enforcers can inspect the plan before execution.
- **Git-diffable**: persisted generated circuits give a reviewable history of plans.
- **Deterministic where possible**: the Manager is stochastic, but the generated circuit is a fixed graph.

The `DelegateNode` interface is the keystone primitive:

```go
type DelegateNode interface {
    Node
    GenerateCircuit(ctx context.Context, nc NodeContext) (*CircuitDef, error)
}
```

## The Kubernetes Operator Analogy

A Kubernetes Operator has a reconciliation loop: watch desired state, observe current state, compute diff, act, loop. An Origami Operator does the same, but the "resources" are circuits and the "desired state" is a goal.

| Kubernetes | Origami Operator |
|---|---|
| Container Runtime (containerd) | `subprocess.Orchestrator` |
| Pod Spec | `CircuitDef` (generated YAML) |
| Running Pod | `CircuitContainer` (active walk) |
| Container in a Pod | Worker Agent (attached to nodes) |
| Volume Mount | `ArtifactScope` (what the Worker can see) |
| Service endpoint | `SignalBus` channel |
| CRD (schema) | `Goal` type |
| CR (instance) | `Goal` instance |
| Operator (controller) | Manager Agent implementing `Operator` |
| HPA (autoscaler) | Dynamic fan-out (add Workers when workload grows) |
| Operator SDK | The Origami Operator API |

### What makes this different from K8s

1. **The topology is cognitive, not infrastructural.** Circuit edges encode cognitive dependencies ("this agent needs that agent's conclusion to reason correctly"), not network dependencies.
2. **The workload is stochastic.** The Operator's `Evaluate` step must assess reasoning quality, not just process health.
3. **Dynamic replanning.** An Origami Operator may change its own plan mid-execution based on what it learns.

## The Reconciliation Loop

```go
type Operator interface {
    Observe(ctx context.Context) (SystemState, error)
    Reconcile(ctx context.Context, goal Goal, state SystemState) (*CircuitDef, error)
    Evaluate(ctx context.Context, goal Goal, result WalkResult) (Evaluation, error)
}

func RunOperator(ctx context.Context, op Operator, goal Goal) error {
    for {
        state, _ := op.Observe(ctx)
        eval, _ := op.Evaluate(ctx, goal, state.LastWalkResult)
        if eval.Met {
            return nil
        }
        if eval.NextAction == ActionEscalate {
            return ErrEscalate{Findings: eval.Findings}
        }
        circuit, _ := op.Reconcile(ctx, goal, state)
        if circuit == nil {
            return nil
        }
        container, _ := runtime.Create(ctx, circuit)
        container.Walk(ctx)
    }
}
```

## The Fractal Property

The hierarchy is self-similar at every level. A Broker is a Manager of Managers. A Manager is a Broker for its Workers. The same four roles repeat at different scopes.

This means one set of primitives (Role, DelegateNode, Scope, FindingRouter) is sufficient — the hierarchy emerges from composition. A "Broker" is just a Manager whose Workers happen to be other Managers. The DSL does not need to know about hierarchy depth; it needs to know about delegation and scoping.

`DelegateNode` is the primitive that enables recursion. A Broker's circuit delegates to Manager circuits. A Manager's circuit delegates to Worker circuits. A deep feature delivery might go three levels; a simple fix goes one. The depth is determined by the problem, not by the framework.

## The Layered Architecture

| Layer | What It Provides | Analog |
|---|---|---|
| **Origami Core** | Circuit engine, graph walk, artifacts, signals | Container runtime |
| **Origami Operator SDK** | Operator interface, reconciliation loop, container runtime, scope engine, finding router | Operator SDK / controller-runtime |
| **Schematics** | Domain-specific operators (RCA Operator, Knowledge Operator, etc.) | Operators built with the SDK (Postgres Operator, Redis Operator) |
| **Origami Central** | Multi-tenant operator registry, cross-circuit coordination, global calibration | Operator Hub / OLM |

Today Origami is at Layer 1 with pieces of Layer 3 (RCA and Knowledge schematics). The Operator SDK (Layer 2) is the bridge.

## Enforcer Feedback Patterns

Three patterns, each suited to a different concern:

### Pattern A: Inline Enforcement (Hook-based)

A Hook runs after a node and can annotate the artifact or return an error to halt the walk. Good for fast, deterministic checks (lint, compile, format). Low overhead.

### Pattern B: Signal-based Enforcement (Asynchronous)

The Enforcer monitors the `SignalBus` and emits findings as signals. The walk's edge conditions can react:

```yaml
edges:
  - from: write_code
    to: revise_code
    when: "signals.has_finding('error')"
```

Good for decoupled enforcement that does not block the walk.

### Pattern C: Parallel Enforcement Circuit

A separate circuit runs concurrently with the work circuit. Both share a signal bus but have independent topologies. Good for deep audits (architecture review, security analysis, performance profiling) that are expensive and should not block the work pipeline.

### Veto Mechanism

Three severity levels with escalating consequences:

- **Info (flag)** — annotate an artifact with a finding. The Manager sees it next planning cycle.
- **Warning (block)** — prevent an artifact from propagating to the next node. Walk pauses.
- **Error (veto)** — abort the sub-circuit. The Manager gets an error artifact and must re-plan. Implemented by overriding artifact confidence to 0, which triggers remand edges in the dialectic.

## Contract Dependency Chain

```
delegate-node (keystone primitive)
├── agent-roles (type system + scoping)
│   └── finding-router (enforcer feedback)
└── operator-reconciliation (reconciliation loop + container runtime)
```

All four contracts are vision-tier in the Origami execution roadmap.

## Information Scoping Model

| Role | Sees | Does not see |
|---|---|---|
| Broker | Project goals, session history, all Manager reports | Individual Worker diffs, Enforcer raw metrics |
| Manager | Its assigned subsystem, Worker artifacts from its Workers | Other Managers' Worker artifacts, Broker session history |
| Worker | Assigned files, task description, relevant context snippets | Other Workers' tasks, full codebase, Manager's plan |
| Enforcer | All artifacts in its audit domain (e.g., all test results) | Worker task assignments, Manager plans, Broker goals |

Implemented via `ArtifactScope` — each artifact is tagged with owner, role, audience, and domain. Each Walker's `NodeContext` is filtered to show only artifacts whose audience includes the Walker's role.

## Calibrating a Hierarchy

### Per-Role Calibration

| Role | What You Measure | How |
|---|---|---|
| Worker | Code quality, task adherence, file scope discipline | Give Worker a scoped task + files, evaluate output against golden diff |
| Manager | Plan quality, decomposition correctness, scope assignment | Give Manager a goal + current state, evaluate generated `CircuitDef` against golden plan |
| Enforcer | Detection rate, false positive rate, finding quality | Give Enforcer known-bad artifacts, measure what it catches |
| Broker | Intent translation accuracy, goal structure quality | Give Broker human utterances, evaluate structured goals against golden goals |

Manager calibration evaluates the generated circuit, not the execution result. You can calibrate a Manager without running any Workers — compare the plan to a golden decomposition.

### Ouroboros for Role Assignment

Ouroboros (metacalibration) answers: "which Persona should fill each Role?" Extend `PersonaSheet` to include role affinity:

- High `persistence` + `convergence_threshold` → Enforcer
- High `speed` + `breadth` → Worker
- High planning-related dimensions → Manager
