# Origami

**Domain-agnostic, YAML-first agentic circuit framework.**

Origami provides graph-based circuit orchestration for AI-powered tools. Define circuits in YAML, build typed node/edge graphs, walk them with AI agents, and observe execution — all through composable primitives that keep domain logic out of the framework.

## Core Primitives

| Primitive | Purpose |
|-----------|---------|
| **Node** | Processing stage — wraps LLM calls, HTTP requests, deterministic transforms, or any computation |
| **Edge** | Conditional connection between nodes — expression-based routing (`when:` via expr-lang) |
| **Walker** | Agent traversing a graph, carrying state and producing artifacts |
| **Transformer** | Pluggable processor invoked at a node (LLM, HTTP, jq, file, or custom) |
| **Hook** | Pre/post step callback for side effects (logging, file writes, enforcement) |
| **Extractor** | Pull structured data from unstructured output |
| **Observer** | Receive walk events for logging, metrics, or UI |

## BYO Architecture

Seven principles govern the boundary between framework and consumer. The invariant: **the framework owns the contract shape, never the implementation.**

| Principle | What you bring | Framework provides |
|-----------|---------------|-------------------|
| **BYOB** — Bots | OpenAI, Anthropic, Ollama, Codex, Claude, custom MCP | `Dispatcher` interface, Stdin/File/Mux/Batch dispatchers |
| **BYOA** — Auth | Vault, OIDC, mTLS, service accounts | Clean injection points on dispatchers and sources |
| **BYOS** — Storage | Postgres, S3, Redis, etcd, CRDs | `curate.Store` interface, `FileStore`, in-memory `WalkerState` |
| **BYOD** — Domain | Your nodes, walkers, hooks, transformers, circuit YAML | Domain-agnostic primitives, built-in transformers (llm, http, jq, file) |
| **BYOR** — Rules | Routing logic, decision thresholds, edge conditions | `when:` expression engine, `EdgeFactory`, `vars:` config |
| **BYO-Data** — Datasets | Ground truth, calibration data, training sets | `curate` package, `ScenarioLoader`/`CaseCollector` interfaces |
| **BYOI** — Infrastructure | Cursor IDE, CLI, Docker Compose, Kubernetes | Topology-agnostic dispatch; same protocol for 4 subagents or 400 pods |

## Quick Start

Define a circuit in YAML:

```yaml
circuit: hello
start: greet
done: summarize
nodes:
  - name: greet
    transformer: llm
    prompt: "Say hello to {{ .input }}"
  - name: summarize
    transformer: llm
    prompt: "Summarize: {{ .prior }}"
edges:
  - from: greet
    to: summarize
```

Run it from Go:

```go
err := framework.Run(ctx, "hello.yaml", "world",
    framework.WithRegistries(registries),
)
```

Or from the CLI:

```bash
origami run hello.yaml --input "world"
```

## Package Structure

```
origami/               Core primitives: Node, Edge, Graph, Walker, CircuitDef
├── calibrate/         Generic calibration harness (ScenarioLoader, CaseCollector, ReportRenderer)
├── connectors/        External system adapters (ReportPortal, GitHub, docs, SQLite)
├── curate/            Dataset curation (Source, Extractor, Store, Record)
├── cycle/             Element interaction rules (generative, destructive)
├── dialectic/         Adversarial debate (thesis/antithesis/synthesis, evidence gaps)
├── dispatch/          Work distribution (SignalBus, MuxDispatcher, FileDispatcher)
├── domainfs/          Virtual filesystem over MCP (MCPRemoteFS)
├── domainserve/       Domain data HTTP server library
├── element/           Behavioral archetypes (Approach, Element, SpeedClass)
├── fold/              Codegen from YAML manifest to domain-serve binary
├── gateway/           Multi-schematic HTTP gateway
├── ingest/            Generic ETL harness (Source, Matcher, DedupStore)
├── kami/              Interactive terminal UI (prompt mode, Sumi TUI)
├── lint/              Circuit YAML linter (structural + binding rules)
├── lsp/               Language Server Protocol support for circuit YAML
├── mask/              Detachable node middleware (Recall, Forge, etc.)
├── mcp/               MCP server for circuit execution (Papercup v2 protocol)
├── models/            LLM model registry
├── ouroboros/          Metacalibration (probes, seeds, persona sheets)
├── persona/           Agent identities (Herald, Seeker, Sentinel, etc.)
├── schematics/        Domain-specific circuit toolkits
│   ├── knowledge/     Knowledge layer (synthesizer, source packs, MCP server)
│   ├── rca/           Root-cause analysis (heuristics, extractors, reports)
│   └── toolkit/       Generic schematic patterns (params, hooks, routing, HITL)
├── subprocess/        Process lifecycle (Orchestrator, ContainerBackend, WorkerPool)
├── sumi/              Diagram rendering (Mermaid, node-state visualization)
├── topology/          Circuit topology validation (cascade, fan-out, fan-in, etc.)
├── transformers/      Built-in transformers (LLM, HTTP, jq, file, match)
└── view/              ViewModel layer for UI consumption
```

## Execution Model

The same circuit YAML and BYO interfaces work across all environments:

| Environment | How it works |
|-------------|-------------|
| **Cursor IDE** | Subagents pull work via MCP (`get_next_step` / `submit_step`) |
| **CLI agent** | `origami run` with Stdin/File dispatcher |
| **Docker Compose** | 4-service topology (Gateway, Engine, Knowledge, Domain) |
| **Kubernetes** | Pod-per-service, same MCP protocol over network |

## Reference Implementations

| Tool | Domain | Repository |
|------|--------|-----------|
| **Asterisk** | Test CI root-cause analysis | `github.com/dpopsuev/asterisk` |
| **Achilles** | AI-driven vulnerability discovery | `github.com/dpopsuev/achilles` |

## License

Internal — Red Hat, Inc.
