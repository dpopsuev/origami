# Contract — Case Study: Cloud-Native Circuit Tools

**Status:** draft  
**Goal:** Analyze five CNCF cloud-native tools against Origami's circuit model, map integration points, identify competitive gaps, and extract actionable tasks to existing contracts.  
**Serves:** System Refinement (nice)

## Contract rules

- The case study document is the primary deliverable. Code changes are secondary.
- Analysis must be evidence-based — cite specific tool features, CNCF maturity, and Origami equivalents.
- Actionable tasks must not duplicate work in existing contracts. Cross-reference explicitly.
- No blind feature copying. Every proposed integration must justify why Origami's existing architecture is the better foundation.
- Classification for each tool: **adapter** (plumbing, no nodes), **marble** (reusable graph node), **infrastructure** (deployment/orchestration), or **no action** (out of scope).

## Context

Cloud-native is an important pillar. Origami circuits run in environments that already have Prometheus, Jaeger, Argo, and Kubernetes. Day-0 integration with this ecosystem is a competitive advantage over Python-based frameworks (CrewAI, LangGraph) that require custom observability wrappers.

### Tools to analyze

| Tool | CNCF Status | Category | Hypothesis |
|------|-------------|----------|------------|
| **Argo Workflows** | Graduated | Workflow orchestration | Origami `CircuitDef` as Argo Workflow template. Long-running production circuit execution on K8s. Infrastructure integration. |
| **Tekton** | Graduated | CI/CD circuits | Relevant for Asterisk's RP integration (analyzing Tekton-triggered failures). Potential Asterisk adapter. |
| **KEDA** | Graduated | Event-driven autoscaling | Scale circuit workers based on queue depth. Pairs with `circuit-efficiency` contract. Infrastructure integration. |
| **Falco** | Graduated | Runtime security | Runtime vulnerability event stream. Relevant for Achilles (vulnerability discovery). Potential Achilles adapter. |
| **CloudEvents** | Graduated | Event format standard | `SignalBus` events as CloudEvents for external consumption. Adapter on SignalBus. |

### Cross-references

- `origami-adapters` — integration recommendations may identify new adapter candidates
- `origami-marbles` — integration recommendations may identify new marble candidates
- `origami-observability` — OTel/Prometheus is the internal implementation; this study covers the broader ecosystem
- `circuit-efficiency` — KEDA autoscaling is related to circuit worker scaling
- `origami-k8s-circuit-operator` (completed design) — Argo analysis builds on the K8s operator design
- `case-study-omo-agentic-arms-race` (completed) — prior competitive analysis pattern
- `case-study-crewai-crews-and-flows` (completed) — prior competitive analysis pattern

## Analysis structure (per tool)

For each tool:

1. **Architecture summary** — what it does, how it works, key abstractions
2. **Concept mapping** — which Origami primitives map to which tool concepts
3. **Gap analysis** — what Origami lacks for integration, what the tool lacks that Origami provides
4. **Integration recommendation** — adapter, marble, infrastructure, or no action
5. **Extracted tasks** — specific tasks cross-referenced to existing contracts

## FSC artifacts

| Artifact | Target | Compartment |
|----------|--------|-------------|
| Cloud-native integration analysis | `docs/case-studies/cloud-native-circuit-tools.md` | domain |
| Integration recommendation matrix | (embedded in analysis) | domain |

## Execution strategy

Phase 1 researches each tool's architecture and API surface. Phase 2 maps concepts to Origami primitives. Phase 3 performs gap analysis. Phase 4 writes integration recommendations with extracted tasks.

## Coverage matrix

| Layer | Applies | Rationale |
|-------|---------|-----------|
| **Unit** | no | Analysis contract — no code |
| **Integration** | no | Analysis contract — no code |
| **Contract** | no | Analysis contract — no code |
| **E2E** | no | Analysis contract — no code |
| **Concurrency** | no | Analysis contract — no code |
| **Security** | no | Analysis contract — no code |

## Tasks

- [ ] **A1** Research Argo Workflows: architecture, Workflow CRD, template types, parameter passing. Map to Origami CircuitDef/Node/Edge.
- [ ] **A2** Research Tekton: Circuit/Task/Step model, Tekton Results, integration with CI triggers. Map to Origami circuit and Asterisk RP integration.
- [ ] **A3** Research KEDA: ScaledObject, triggers, scaling policies. Map to Origami dispatch worker scaling.
- [ ] **A4** Research Falco: rules engine, event stream, Falco Sidekick. Map to Achilles vulnerability event consumption.
- [ ] **A5** Research CloudEvents: spec, Go SDK, transport bindings. Map to Origami SignalBus event format.
- [ ] **A6** Write per-tool analysis (5 sections) following the structure above.
- [ ] **A7** Write integration recommendation matrix: tool × classification × priority × target contract.
- [ ] **A8** Extract actionable tasks and inject into target contracts as cross-references.

## Acceptance criteria

**Given** the completed analysis,  
**When** each tool has an integration recommendation,  
**Then** the recommendation is one of: adapter (with FQCN), marble (with FQCN), infrastructure (with target contract), or no action (with rationale).

**Given** the extracted tasks,  
**When** they reference existing contracts,  
**Then** each task is specific enough to be added as a phase or task in the target contract without further research.

## Security assessment

No trust boundaries affected. This is a research contract.

## Notes

2026-02-26 — Contract created. Motivated by the question "Could we treat metrics and monitoring as first-class citizens with day-0 Prometheus integration?" The `origami-observability` contract handles the internal OTel/Prometheus implementation. This case study covers the broader cloud-native ecosystem: Argo (production orchestration), Tekton (CI/CD), KEDA (autoscaling), Falco (security), CloudEvents (event format). All five are CNCF Graduated — stable, widely adopted, and relevant to Red Hat's telco QE mission on OpenShift.
