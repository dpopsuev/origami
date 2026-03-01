# Case Study: Cloud-Native Circuit Tools — CNCF Ecosystem Integration Analysis

**Date:** 2026-02-26  
**Subject:** Five CNCF Graduated tools analyzed against Origami's circuit model  
**Source:** `argoproj.github.io/argo-workflows`, `tekton.dev`, `keda.sh`, `falco.org`, `cloudevents.io`  
**Purpose:** Map each tool's architecture to Origami primitives, classify integration type (adapter/marble/infrastructure/no-action), identify competitive gaps, and extract actionable tasks to existing contracts.

---

## Executive Summary

Origami circuits run in environments where Argo, Tekton, KEDA, Falco, and CloudEvents are already present. Unlike Python-based agentic frameworks (CrewAI, LangGraph) that require custom integration wrappers, Origami's Go-native, Kubernetes-native design enables day-0 interoperability with this ecosystem. This analysis covers five CNCF Graduated tools and produces a concrete integration matrix.

| Tool | Classification | Priority | Rationale |
|------|---------------|----------|-----------|
| **Argo Workflows** | Infrastructure | High | Production execution of Origami circuits on K8s |
| **Tekton** | Adapter | Medium | Source of CI failure data for Asterisk RP integration |
| **KEDA** | Infrastructure | Medium | Auto-scale circuit dispatch workers |
| **Falco** | Adapter | Medium | Runtime security event source for Achilles |
| **CloudEvents** | Adapter | Low | Standard event envelope for SignalBus externalization |

---

## 1. Argo Workflows

### 1.1 Architecture Summary

Argo Workflows is the CNCF Graduated workflow engine for Kubernetes. It defines workflows as Kubernetes CRDs, with each step running in its own Pod.

**Key abstractions:**

- **Workflow**: top-level resource containing a DAG or steps definition
- **Template**: a reusable unit of work (container, script, DAG, or steps)
- **DAG template**: defines tasks as a directed acyclic graph with dependency edges
- **Steps template**: defines sequential/parallel execution stages
- **Parameters**: typed inputs/outputs passed between templates
- **Artifacts**: files produced by one step and consumed by another (S3, GCS, MinIO)
- **Workflow Controller**: reconciliation loop processing Workflow resources
- **Argo Server**: HTTP API + UI for submission, monitoring, artifact browsing

**Pod structure:** Each workflow step spawns a Pod with three containers: `init` (fetch artifacts), `main` (user workload), `wait` (save artifacts and parameters). The Controller processes one Workflow at a time per worker goroutine.

### 1.2 Concept Mapping

| Argo Concept | Origami Equivalent | Alignment |
|-------------|-------------------|-----------|
| Workflow | `CircuitDef` | Strong — both are declarative graph definitions |
| DAG template | `CircuitDef.Nodes` + `CircuitDef.Edges` | Strong — DAG tasks are nodes, dependencies are edges |
| Steps template | N/A (Origami uses edges for sequencing) | Partial — steps are a special case of a linear DAG |
| Template | `Node` (process) or `Marble` (composite) | Strong — Argo templates are reusable units, like Marbles |
| Parameters | `WalkerState.Context` | Partial — Origami passes context through walker state, not typed parameters |
| Artifacts | `Artifact` interface | Partial — Origami artifacts are in-process objects, not file-based |
| Workflow Controller | `Walker` / `Run()` | Structural — the controller drives execution; Origami's walker walks the graph |
| Argo Server | Kami | Partial — both serve APIs and UIs for observability |
| Retry/backoff | N/A | Gap — Origami has no built-in retry on node failure |

### 1.3 Gap Analysis

**Origami lacks:**
- **Durable execution across Pod restarts.** Argo workflows survive node evictions because state lives in the Kubernetes API server. Origami walks are in-process. The `durable-execution` contract addresses this with Checkpointer + HITL Interrupt/Resume.
- **Typed parameter passing between nodes.** Argo has first-class parameter schemas with JSON Schema validation. Origami uses `WalkerState.Context` (untyped `map[string]any`). Adding typed parameters would improve validation.
- **File artifact management.** Argo integrates with S3/GCS/MinIO for artifact persistence between steps. Origami artifacts are in-memory and lost between sessions. The `durable-execution` contract's Checkpointer partially addresses this.
- **Retry policies.** Argo supports `retryStrategy` with limit, backoff, and expression-based conditions. Origami has loop edges but no retry-on-failure semantics.

**Argo lacks:**
- **AI-native abstractions.** No concept of Element, Persona, Mask, or Adversarial Dialectic. Argo is infrastructure; Origami is intelligence.
- **In-process circuit execution.** Every Argo step requires a Pod (seconds of overhead). Origami walks execute in microseconds because nodes are in-process function calls.
- **Dynamic edge evaluation.** Argo DAG edges are static dependencies. Origami edges evaluate `when:` expressions at runtime against artifacts, enabling dynamic routing.
- **Metacalibration.** No equivalent to Ouroboros for measuring and improving circuit quality.

### 1.4 Integration Recommendation: **Infrastructure**

Argo Workflows is the execution substrate for production Origami circuits. The integration path:

1. **Origami-to-Argo compiler**: Translate `CircuitDef` to Argo Workflow CRD YAML. Each Origami Node becomes an Argo container template. Each Origami Edge becomes an Argo DAG dependency. `when:` expressions become Argo conditional steps.
2. **Argo step container image**: A single Docker image containing the `origami` binary that executes a single Node's `Process()` function. Parameters marshalled as JSON.
3. **Artifact bridge**: Map Origami `Artifact` interface to Argo artifact store (S3). Serialize artifacts to JSON between steps.

This is not an adapter (no external data source) or a marble (not a reusable graph node). It's infrastructure — the deployment model for production Origami circuits.

### 1.5 Extracted Tasks

- **Target: `durable-execution`** — Argo integration depends on durable state. The Checkpointer design must support artifact serialization/deserialization across process boundaries. Add: "Argo artifact bridge: serialize/deserialize Artifact to JSON for inter-step transfer."
- **Target: `circuit-efficiency`** — Argo step overhead (Pod creation) makes node caching more valuable. Add: "Argo cost model: estimate per-step Pod overhead to prioritize caching over re-execution."
- **Target: new contract `origami-argo-compiler`** — The Argo integration is large enough for its own contract: CircuitDef → Argo Workflow CRD compilation, container image build, artifact bridge.

---

## 2. Tekton

### 2.1 Architecture Summary

Tekton is the CNCF Graduated cloud-native CI/CD framework. Like Argo, it runs on Kubernetes as CRDs, but is purpose-built for CI/CD rather than general-purpose workflows.

**Key abstractions:**

- **Step**: the smallest execution unit — a container running a specific action
- **Task**: a collection of sequential Steps, accepting parameters and producing results
- **TaskRun**: an instantiation of a Task
- **Circuit**: a collection of Tasks with dependency edges and shared Workspaces
- **CircuitRun**: an instantiation of a Circuit
- **Workspace**: shared volume for data exchange between Tasks
- **Trigger / EventListener**: webhook-based event ingestion that creates CircuitRuns
- **Tekton Results**: API for querying completed CircuitRun/TaskRun outcomes

**Execution model:** Each Task runs as a Pod; each Step runs as a container within that Pod. Circuits compose Tasks with `runAfter` dependencies. Tekton Catalog provides a registry of reusable Tasks.

### 2.2 Concept Mapping

| Tekton Concept | Origami Equivalent | Alignment |
|---------------|-------------------|-----------|
| Circuit | `CircuitDef` | Strong — both are declarative circuit definitions |
| Task | `Node` | Strong — a Task is a unit of work with inputs/outputs |
| Step | N/A (Origami nodes are atomic) | Partial — Steps are sub-node granularity |
| CircuitRun | `Walker` walk session | Structural — both represent a circuit execution instance |
| Workspace | `WalkerState.Context` | Partial — both share data between tasks/nodes |
| Trigger | N/A (Origami is pull-based) | Gap — Origami has no event-driven triggering |
| Tekton Results | `MemoryStore` / calibration report | Partial — both store execution outcomes |
| Tekton Catalog | `MarbleRegistry` + `AdapterManifest` | Structural — both are registries of reusable components |

### 2.3 Gap Analysis

**Origami lacks:**
- **Event-driven triggering.** Tekton's EventListeners create CircuitRuns on webhook events. Origami circuits are explicitly invoked. For production use (Asterisk analyzing failures as they happen), an event-driven trigger mechanism is needed.
- **Results API for querying past runs.** Tekton Results provides a searchable history of circuit executions. Origami's MemoryStore and calibration reports serve a similar purpose but lack a query API.

**Tekton lacks:**
- **AI-native circuit semantics.** No LLM transformers, no element-persona model, no confidence scoring.
- **Dynamic routing.** Tekton `when` expressions are limited to input comparisons. Origami `when:` expressions can evaluate against full artifact state.
- **Metacalibration.** No mechanism to measure and improve circuit quality over time.

### 2.4 Integration Recommendation: **Adapter**

Tekton is relevant to Asterisk as a **data source**, not as an execution platform. Asterisk analyzes failures from CI circuits — many of which are Tekton Circuits running on OpenShift.

**Proposed adapter: `tekton.results`**

- **FQCN:** `tekton.results`
- **Type:** Adapter (data source)
- **Function:** Query Tekton Results API for CircuitRun/TaskRun data, extract failure logs, map to Asterisk's FailureItem format
- **Consumer:** Asterisk (`consume` circuit)

This enables Asterisk to directly ingest Tekton CI failures without requiring ReportPortal as an intermediary, broadening the data source portfolio.

### 2.5 Extracted Tasks

- **Target: `origami-adapters`** — Add adapter candidate: `tekton.results` (Tekton Results API client). FQCN `tekton.results`, provides transformers for TaskRun/CircuitRun log extraction and failure classification.
- **Target: Asterisk `consume` circuit** — Tekton as an alternative to RP for CI failure ingestion. The consume circuit would gain a `tekton-discover` node alongside the existing `rp-discover` node, with a routing edge based on configured source type.

---

## 3. KEDA

### 3.1 Architecture Summary

KEDA (Kubernetes Event-Driven Autoscaling) extends the Kubernetes HPA to scale based on external event sources rather than CPU/memory.

**Key abstractions:**

- **ScaledObject**: CRD binding a Deployment/StatefulSet to one or more triggers
- **ScaledJob**: CRD creating Jobs from event sources (queue-based)
- **Trigger / Scaler**: a plugin that reads metrics from an external source (70+ built-in: Kafka, RabbitMQ, Prometheus, AWS SQS, etc.)
- **Metrics Server**: bridges external metrics into the Kubernetes metrics API
- **KEDA Operator**: reconciles ScaledObjects, manages HPA lifecycle, handles scale-to-zero

**Execution model:** KEDA polls external metrics via scalers, exposes them as Kubernetes external metrics, and the standard HPA uses these metrics to scale workloads. Scale-to-zero is achieved by the KEDA Operator directly (HPA cannot scale below 1).

### 3.2 Concept Mapping

| KEDA Concept | Origami Equivalent | Alignment |
|-------------|-------------------|-----------|
| ScaledObject | N/A | Gap — Origami has no autoscaling abstraction |
| Trigger (scaler) | N/A | Gap — Origami has no external metric ingestion |
| Scale-to-zero | `dispatch.Dispatcher` idle behavior | Weak — dispatchers don't manage their own scaling |
| ScaledJob | Parallel walker count in `Run()` | Weak — `--parallel` is static, not event-driven |
| Prometheus scaler | `origami-observability` Prometheus endpoint | Potential — KEDA could read Origami's `/metrics` to scale workers |

### 3.3 Gap Analysis

**Origami lacks:**
- **Dynamic worker scaling.** Origami's `--parallel` flag sets a fixed worker count. In production, case queue depth varies. KEDA can scale workers from 0 to N based on queue length.
- **Kubernetes-native deployment model.** Origami circuits run as local processes. KEDA requires Kubernetes Deployments/Jobs to scale. This depends on the Argo integration (Section 1).

**KEDA lacks:**
- Everything Origami provides. KEDA is pure infrastructure — it scales existing workloads but has no concept of circuits, AI, or analysis.

### 3.4 Integration Recommendation: **Infrastructure**

KEDA is the scaling layer for production Origami circuit workers. It pairs with the Argo integration:

1. Argo compiles CircuitDef to Workflow CRDs
2. Worker Pods execute Node's Process() functions
3. KEDA scales worker replicas based on queue depth (e.g., number of pending cases)

The metric source for KEDA could be:
- **Prometheus**: Origami's `/metrics` endpoint exposes `origami_cases_pending` gauge
- **Message queue**: If dispatch evolves to use NATS/Kafka, KEDA scales on topic lag

No Origami code changes are needed beyond the observability contract's Prometheus endpoint.

### 3.5 Extracted Tasks

- **Target: `origami-observability`** — Ensure the Prometheus endpoint exposes metrics that KEDA scalers can consume: `origami_cases_pending`, `origami_workers_active`, `origami_dispatch_queue_depth`. Add: "KEDA-compatible metrics: expose queue depth and worker count as Prometheus gauges."
- **Target: `circuit-efficiency`** — KEDA dynamic scaling is the production answer to the `--parallel` flag. Add: "Document KEDA ScaledObject template for Origami worker Deployments."

---

## 4. Falco

### 4.1 Architecture Summary

Falco is the CNCF Graduated runtime security engine. It detects threats by monitoring kernel-level events via eBPF probes and evaluating them against YAML detection rules.

**Key abstractions:**

- **eBPF probe / kernel module**: intercepts system calls from container workloads
- **Rules engine**: YAML-based detection rules with conditions, output templates, and priority levels
- **Event enrichment**: raw kernel events are enriched with Kubernetes metadata (pod name, namespace, labels, service account)
- **Falcosidekick**: alert router that forwards Falco events to Slack, Kafka, Elasticsearch, S3, webhooks, and 50+ destinations
- **Plugin architecture**: extends Falco beyond kernel monitoring to cloud APIs (CloudTrail, GitHub audit logs, Okta)

**Detection model:** Falco runs as a DaemonSet on every Kubernetes node. When a syscall matches a rule condition, Falco emits an alert with enriched metadata. Falcosidekick routes alerts to external systems.

### 4.2 Concept Mapping

| Falco Concept | Origami Equivalent | Alignment |
|--------------|-------------------|-----------|
| Detection rule | `CircuitDef` edge `when:` expression | Structural — both evaluate conditions against events |
| Falcosidekick | `SignalBus` | Structural — both route events to external consumers |
| Plugin | `Adapter` / `Extractor` | Structural — both extend a core engine with external data sources |
| Alert priority (emergency → debug) | `Severity` (Critical → Low) | Structural — both classify events by severity |
| Event enrichment | `Transformer` chain | Structural — both enrich raw data with context |

### 4.3 Gap Analysis

**Origami lacks:**
- **Streaming event consumption.** Falco produces a continuous stream of security events. Origami circuits are request/response (walk a graph, produce an artifact). Streaming event ingestion requires either a polling adapter or a push-based trigger.
- **Alert aggregation.** Falco events are high-volume (thousands per minute in a busy cluster). Origami would need deduplication and windowing before feeding events into a circuit.

**Falco lacks:**
- **Root-cause analysis.** Falco detects threats but doesn't analyze them. It says "unexpected shell in container X" but doesn't determine whether the shell is from a compromised dependency, a misconfigured deployment, or an intentional debugging session.
- **Vulnerability correlation.** Falco doesn't connect runtime behavior to known CVEs. Achilles provides this correlation.

### 4.4 Integration Recommendation: **Adapter**

Falco is a data source for Achilles. Runtime security events complement Achilles' static vulnerability scanning with behavioral evidence.

**Proposed adapter: `falco.events`**

- **FQCN:** `falco.events`
- **Type:** Adapter (event source)
- **Function:** Consume Falco events via Falcosidekick webhook, deduplicate and window, feed into an Achilles circuit as a `runtime-evidence` node
- **Consumer:** Achilles (vulnerability assessment)

The integration enables Achilles to correlate static vulnerability findings with runtime behavioral evidence: "CVE-2025-XXXX is present in dependency Y, AND Falco detected unexpected network activity from containers running that dependency."

### 4.5 Extracted Tasks

- **Target: `origami-adapters`** — Add adapter candidate: `falco.events` (Falcosidekick webhook consumer). FQCN `falco.events`, provides extractors for Falco alert parsing and severity mapping.
- **Target: Achilles circuit** — Add optional `runtime-evidence` node after `scan` that consumes Falco events when available, correlating static findings with runtime behavior. This is a post-PoC enhancement.

---

## 5. CloudEvents

### 5.1 Architecture Summary

CloudEvents is the CNCF Graduated specification for describing events in a common format. It defines a standard envelope (metadata + payload) that any event producer or consumer can understand.

**Key abstractions:**

- **Event**: a structured envelope with required attributes (`id`, `source`, `type`, `specversion`) and optional attributes (`subject`, `time`, `datacontenttype`, `data`)
- **Protocol binding**: mapping of CloudEvents to transport protocols (HTTP, Kafka, AMQP, NATS, MQTT)
- **Format encoding**: JSON (mandatory) or Protobuf/Avro (optional)
- **Producer / Consumer / Intermediary**: standard roles in event flow
- **Go SDK** (`cloudevents/sdk-go`): typed client for producing and consuming events with protocol adapters

**Design philosophy:** CloudEvents solves the "every system defines events differently" problem. By standardizing the envelope, systems can route, filter, and process events without understanding the payload.

### 5.2 Concept Mapping

| CloudEvents Concept | Origami Equivalent | Alignment |
|--------------------|-------------------|-----------|
| Event envelope | `KamiEvent` / `WalkEvent` | Strong — both are structured event types with metadata |
| `source` attribute | `WalkEvent.Node` or `WalkEvent.Edge` | Partial — Origami events have source context but not URI-based |
| `type` attribute | `WalkEvent.Type` (EventNodeEnter, EventNodeExit, etc.) | Strong — both categorize events |
| `subject` attribute | `WalkEvent.Walker` or case ID | Partial |
| Protocol binding | `SignalBus`, WebSocket, SSE | Structural — both abstract transport from event format |
| Go SDK | `origami/kami` + `origami/observability` | Structural — both are Go libraries for event handling |

### 5.3 Gap Analysis

**Origami lacks:**
- **Standard event format.** Origami's `WalkEvent` and `KamiEvent` are internal types with no external standard. External consumers (monitoring dashboards, CI systems, SIEM tools) would need custom parsers.
- **Protocol bindings.** SignalBus uses WebSocket and SSE. CloudEvents supports HTTP, Kafka, AMQP, NATS, and MQTT. Supporting CloudEvents protocol bindings would unlock integration with any system that speaks CloudEvents.

**CloudEvents lacks:**
- Everything Origami provides. CloudEvents is a data format, not a framework. It standardizes the envelope; Origami provides the intelligence.

### 5.4 Integration Recommendation: **Adapter**

CloudEvents is a serialization layer for Origami's event system. The integration is thin: wrap existing events in CloudEvents envelopes for external consumption.

**Proposed adapter: `cloudevents.bridge`**

- **FQCN:** `cloudevents.bridge`
- **Type:** Adapter (event format)
- **Function:** Wrap `WalkEvent` / `KamiEvent` in CloudEvents envelopes; publish via CloudEvents Go SDK to any supported transport (HTTP, Kafka, NATS)
- **Consumer:** All Origami-based tools (framework-level)

Example CloudEvent from an Origami circuit walk:
```json
{
  "specversion": "1.0",
  "type": "io.origami.walk.node.exit",
  "source": "/circuit/asterisk-rca/node/triage",
  "id": "a1b2c3d4",
  "time": "2026-02-26T10:30:00Z",
  "datacontenttype": "application/json",
  "data": {
    "walker": "w-001",
    "node": "triage",
    "elapsed_ms": 1234,
    "artifact_type": "triage",
    "confidence": 0.92
  }
}
```

### 5.5 Extracted Tasks

- **Target: `origami-observability`** — The CloudEvents adapter lives in the observability layer alongside OTel and Prometheus. Add: "CloudEvents exporter: optional `WalkObserver` implementation that serializes events to CloudEvents format."
- **Target: `origami-adapters`** — Add adapter candidate: `cloudevents.bridge`. Register in core adapter as optional module.

---

## Integration Matrix

| Tool | Classification | FQCN | Priority | Target Contract | Consumer |
|------|---------------|------|----------|----------------|----------|
| **Argo Workflows** | Infrastructure | N/A (compiler) | High | New: `origami-argo-compiler` | All |
| **Tekton** | Adapter | `tekton.results` | Medium | `origami-adapters` | Asterisk |
| **KEDA** | Infrastructure | N/A (deployment) | Medium | `origami-observability`, `circuit-efficiency` | All |
| **Falco** | Adapter | `falco.events` | Medium | `origami-adapters` | Achilles |
| **CloudEvents** | Adapter | `cloudevents.bridge` | Low | `origami-observability`, `origami-adapters` | All |

## Competitive Positioning

The cloud-native integration story gives Origami a structural advantage over Python-based agentic frameworks:

| Capability | Origami | CrewAI | LangGraph |
|-----------|---------|--------|-----------|
| Kubernetes-native execution | Go binary → container → Argo Workflow | Python process + custom K8s wrappers | Python process + custom K8s wrappers |
| Prometheus metrics | `DefaultObservability()` + `/metrics` | Custom instrumentation required | Custom instrumentation required |
| KEDA autoscaling | Native via Prometheus metrics | Requires custom metric adapters | Requires custom metric adapters |
| Falco integration | Go SDK, direct eBPF event parsing | Python gRPC client required | Python gRPC client required |
| CloudEvents | Go SDK, native protocol bindings | Python SDK, limited bindings | Python SDK, limited bindings |
| Container image size | ~20MB (static Go binary) | ~500MB+ (Python + dependencies) | ~500MB+ (Python + dependencies) |
| Cold start latency | <100ms | 2-5s (Python interpreter + imports) | 2-5s (Python interpreter + imports) |

The Go-native + Kubernetes-native combination means Origami integrations are thin adapters rather than heavyweight bridge services. This aligns with Red Hat's telco QE mission where OpenShift is the execution platform and every additional Pod is a cost.

## Summary of Extracted Tasks

| # | Task | Target Contract | From |
|---|------|----------------|------|
| 1 | Argo artifact bridge: serialize/deserialize Artifact across process boundaries | `durable-execution` | Argo |
| 2 | Argo cost model: per-step Pod overhead vs caching benefit | `circuit-efficiency` | Argo |
| 3 | New contract: `origami-argo-compiler` (CircuitDef → Argo Workflow CRD) | New | Argo |
| 4 | Adapter candidate: `tekton.results` (Tekton Results API client) | `origami-adapters` | Tekton |
| 5 | Tekton as alternative RP source in Asterisk consume circuit | Asterisk | Tekton |
| 6 | KEDA-compatible metrics: expose queue depth and worker count as Prometheus gauges | `origami-observability` | KEDA |
| 7 | Document KEDA ScaledObject template for Origami worker Deployments | `circuit-efficiency` | KEDA |
| 8 | Adapter candidate: `falco.events` (Falcosidekick webhook consumer) | `origami-adapters` | Falco |
| 9 | Optional `runtime-evidence` node in Achilles circuit | Achilles | Falco |
| 10 | CloudEvents exporter: WalkObserver serializing to CloudEvents format | `origami-observability` | CloudEvents |
| 11 | Adapter candidate: `cloudevents.bridge` | `origami-adapters` | CloudEvents |
