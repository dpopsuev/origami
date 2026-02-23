# Contract — origami-dsl-transformers

**Status:** complete  
**Goal:** Nodes declare a `transformer:` in YAML; the framework calls the transformer directly, eliminating the Walker + Node.Process() indirection for the common case. Built-in transformers cover LLM, HTTP, file, and jq.  
**Serves:** Origami DSL

## Contract rules

Global rules only, plus:

- **Part of a 5-contract series.** Contract 2 of 5. Depends on `origami-dsl-expression-engine` (C1). Required by `origami-dsl-pipeline-vars` (C3).
- **Extractor evolves, not replaced.** The existing `Extractor` interface is the foundation. `Transformer` may be a rename or a wrapper adding schema validation and data flow. Design decision made during execution.
- **Walker remains as escape hatch.** For nodes that need custom walk logic (like Ansible's `raw` module), `Walker.Handle()` still works. The `transformer:` path is the default, not the only option.
- **Built-in transformers are batteries-included.** They require zero custom code. Domain transformers implement the same interface.

## Context

- `origami-dsl-expression-engine` — Predecessor. Provides real `when:` evaluation so edges route based on transformer output.
- `extractor.go` — Current `Extractor` interface: `Extract(ctx, input) (output, error)`. `extractorNode` already delegates to extractors.
- `runner.go` — `Runner.Walk()` wraps walker with schema validation.
- `walker.go` — `Walker` interface: `Handle(ctx, Node, NodeContext) (Artifact, error)`.
- `asterisk/internal/calibrate/walker.go` — `calibrationWalker` that sends prompts, parses artifacts, extracts metrics.

## FSC artifacts

| Artifact | Target | Compartment |
|----------|--------|-------------|
| Transformer pattern reference | `docs/` | domain |
| Built-in transformer catalog | `docs/` | domain |

## Execution strategy

Phase 1: Design `Transformer` interface and `TransformerRegistry`. Phase 2: Implement built-in transformers (`llm`, `http`, `jq`, `file`). Phase 3: Update `BuildGraph` and `Walk` to invoke transformers directly. Phase 4: Migrate Asterisk from Walker-based to transformer-based. Phase 5: Validate, tune, validate.

## Coverage matrix

| Layer | Applies | Rationale |
|-------|---------|-----------|
| **Unit** | yes | Each built-in transformer: happy path, error cases, edge inputs |
| **Integration** | yes | Pipeline walk invokes transformer, validates schema, evaluates edges |
| **Contract** | yes | `Transformer` interface compliance across all implementations |
| **E2E** | yes | Asterisk stub calibration via transformer-based nodes |
| **Concurrency** | no | Transformers are stateless |
| **Security** | yes | `http` transformer: SSRF risk. `file` transformer: path traversal. `llm` transformer: prompt injection. |

## Tasks

All tasks complete.

## Acceptance criteria

**Given** the Origami framework with `NodeDef.Transformer` support,  
**When** a pipeline YAML declares `transformer: llm` and `prompt: prompts/recall.md` on a node,  
**Then**:
- The framework resolves the transformer from the registry at build time
- The framework calls the transformer directly during Walk (no Walker.Handle required)
- Built-in `llm`, `http`, `jq`, `file` transformers are available with zero custom code
- `Walker` still works as escape hatch for nodes without `transformer:`
- Asterisk stub calibration produces identical metrics
- `go build ./...` and `go test ./...` pass in Origami, Asterisk, and Achilles

## Security assessment

| OWASP | Finding | Mitigation |
|-------|---------|------------|
| A03 Injection | `llm` transformer renders prompt templates with user-controlled data. Potential prompt injection. | Template rendering uses text/template (no HTML escaping issues). Prompt content is domain-controlled, not user-facing. |
| A10 SSRF | `http` transformer makes outbound requests to URLs specified in YAML. | URL allowlist validation. Only HTTPS by default. Configurable allowed hosts. |
| A01 Path Traversal | `file` transformer reads from disk paths specified in YAML. | Path must be relative to pipeline directory. Reject `..` components. Configurable root directory. |

## Notes

2026-02-23 12:00 — Contract created. Depends on `origami-dsl-expression-engine`. The `Extractor` interface (Tome V) is the foundation; this contract evolves it into a broader `Transformer` primitive with built-in implementations.

2026-02-18 — Contract completed as part of DSL C1-C5 initiative. All phases executed, green gates passed across Origami, Asterisk, and Achilles.
