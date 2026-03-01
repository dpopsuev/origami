# Contract — origami-dsl-expression-engine

**Status:** complete  
**Goal:** `EdgeDef.When` expressions are evaluated by the framework at runtime against artifact JSON and walker state, eliminating Go closure edge factories for the common case.  
**Serves:** Origami DSL

## Contract rules

Global rules only, plus:

- **Part of a 5-contract series.** This is contract 1 of 5 in the Origami DSL initiative. Contracts 2-5 depend on this one. See plan: "Origami DSL -- 5 Contracts for Deterministic Circuits on Nondeterministic Data."
- **Expression engine is a dependency, not a feature.** The expression library (`expr-lang/expr`) is an implementation detail. The public API is the `when:` field on `EdgeDef`.
- **EdgeFactory remains as escape hatch.** Edges with `when:` use expression evaluation. Edges without `when:` but with a registered `EdgeFactory` entry use Go closures. Edges with neither always match (existing `dslEdge` behavior). This preserves backward compatibility.
- **Soul:** "Once a node transforms unstructured into structured, the routing becomes deterministic because you're evaluating against typed fields. The LLM is probabilistic; the circuit is not."

## Context

- `origami-agentic-network-framework` — Predecessor. Established Extractor[In, Out] as Tome V. Complete.
- `distillation-endgame` (cross-repo) — Moved dispatch, logging, format, workspace to Origami. Built Runner.Walk(), artifact schemas. Complete.
- `dsl.go` — Current DSL: `CircuitDef`, `NodeDef`, `EdgeDef`, `BuildGraph`, `dslEdge`.
- `edge.go` — `Edge` interface with `Evaluate(Artifact, *WalkerState) *Transition`.
- `graph.go` — `DefaultGraph.Walk()` calls `edge.Evaluate()` for routing.
- `asterisk/internal/orchestrate/heuristics.go` — 17 Go closures that will become `when:` expressions.
- `asterisk/circuits/rca-investigation.yaml` — Existing circuit YAML with comment-only `condition:` fields.

## FSC artifacts

| Artifact | Target | Compartment |
|----------|--------|-------------|
| Expression engine design reference | `docs/` | domain |
| Updated DSL principles (expression evaluation) | `rules/domain/` | universal |
| `when:` field glossary entry | `glossary/` | domain |

## Execution strategy

Phase 1: Add `expr-lang/expr` dependency and implement `expressionEdge`. Phase 2: Update `BuildGraph` to create `expressionEdge` when `When` is set. Phase 3: Migrate Asterisk heuristics to `when:` expressions and validate identical behavior. Phase 4: Validate, tune, validate.

## Coverage matrix

| Layer | Applies | Rationale |
|-------|---------|-----------|
| **Unit** | yes | Expression compilation, evaluation against various contexts, error messages |
| **Integration** | yes | `BuildGraph` with `when:` edges; full circuit walk with expression-only routing |
| **Contract** | yes | `EdgeDef.When` field accepted in YAML; `expressionEdge` implements `Edge` interface |
| **E2E** | yes | Asterisk stub calibration with expression-only edges produces identical metrics |
| **Concurrency** | no | Compiled expressions are immutable and safe for concurrent evaluation |
| **Security** | yes | Expression injection: user-provided YAML could contain malicious expressions |

## Tasks

All tasks complete.

## Acceptance criteria

**Given** the Origami framework with `EdgeDef.When` support,  
**When** a circuit YAML declares `when: "output.match == true && output.confidence >= config.recall_hit"` on an edge,  
**Then**:
- The expression compiles at graph build time (invalid expressions fail with clear error)
- The expression evaluates at walk time against `{output, state, config}` context
- `state.loops.investigate` is accessible in expressions for loop-count checks
- All 17 Asterisk heuristics are expressible as `when:` conditions
- `EdgeFactory` still works as escape hatch for edges without `when:`
- Asterisk stub calibration produces identical metrics (zero regression)
- `go build ./...` and `go test ./...` pass in Origami, Asterisk, and Achilles

## Security assessment

| OWASP | Finding | Mitigation |
|-------|---------|------------|
| A03 Injection | `when:` expressions come from YAML files. A malicious YAML could contain expressions that access unexpected fields or cause resource exhaustion. | `expr-lang/expr` is non-Turing-complete (no loops, no assignments, always terminates). Expressions are compiled with a declared environment limiting accessible fields. No file I/O or network access from expressions. |
| A05 Misconfiguration | Verbose expression compilation errors could leak internal type information. | Wrap compilation errors to strip internal details. |

## Notes

2026-02-23 12:00 — Contract created. Gate contract for the Origami DSL initiative (5 contracts). Expression engine `expr-lang/expr` selected: lightweight, Go-native, non-Turing-complete, 5.8k stars, actively maintained. Alternatives considered: `google/cel-go` (too heavy, protobuf dependency), custom evaluator (maintenance risk).

2026-02-18 — Contract completed as part of DSL C1-C5 initiative. All phases executed, green gates passed across Origami, Asterisk, and Achilles.
