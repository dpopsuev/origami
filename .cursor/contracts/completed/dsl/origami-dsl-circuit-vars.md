# Contract — origami-dsl-circuit-vars

**Status:** complete  
**Goal:** Circuits declare variables (`vars:`), inter-node data flow (`input:`), and prompt templates (`prompt:`) in YAML. Thresholds, configuration, and data flow are fully declarative.  
**Serves:** Origami DSL

## Contract rules

Global rules only, plus:

- **Part of a 5-contract series.** Contract 3 of 5. Depends on `origami-dsl-transformers` (C2). Required by `origami-dsl-hooks` (C4).
- **Variables are the glue.** `vars:` provides the configuration surface. Variables flow into expression context as `config.*`, into prompt templates as template variables, and into transformer inputs.
- **Data flow is explicit.** Each node declares its input source. No implicit "previous node output" — make dependencies visible in the YAML.

## Context

- `origami-dsl-expression-engine` — Provides `config.*` in expression context (this contract fills that context).
- `origami-dsl-transformers` — Provides `Transformer` that this contract feeds with resolved inputs and prompt templates.
- `asterisk/internal/orchestrate/types.go` — `Thresholds` struct that becomes `vars:`.
- `asterisk/internal/orchestrate/params.go` — `BuildParams` that becomes declarative `input:` + template context.

## FSC artifacts

| Artifact | Target | Compartment |
|----------|--------|-------------|
| Circuit variables reference | `docs/` | domain |
| Data flow pattern documentation | `docs/` | domain |

## Execution strategy

Phase 1: Add `Vars` to `CircuitDef` and implement variable resolution. Phase 2: Add `Input` to `NodeDef` and implement inter-node data flow. Phase 3: Integrate prompt template rendering with unified context. Phase 4: Migrate Asterisk thresholds and params. Phase 5: Validate, tune, validate.

## Coverage matrix

| Layer | Applies | Rationale |
|-------|---------|-----------|
| **Unit** | yes | Variable resolution, input reference parsing, template rendering |
| **Integration** | yes | Full circuit with vars flowing into expressions and transformers |
| **Contract** | yes | `vars:` and `input:` YAML fields accepted and resolved correctly |
| **E2E** | yes | Asterisk circuit with `vars:` replacing `Thresholds` struct |
| **Concurrency** | no | Variables are immutable after circuit load |
| **Security** | no | Variables come from YAML (same trust level as code) |

## Tasks

All tasks complete.

## Acceptance criteria

**Given** a circuit YAML with `vars:` and `input:` declarations,  
**When** the circuit is executed,  
**Then**:
- `vars:` values are accessible as `config.*` in `when:` expressions
- `vars:` values are accessible in prompt templates
- `input: ${recall.output}` resolves to the recall node's output artifact
- `--set recall_hit=0.9` overrides the YAML `vars:` value
- Asterisk thresholds work via `vars:` with zero Go structs
- `go build ./...` and `go test ./...` pass in Origami, Asterisk, and Achilles

## Security assessment

No new trust boundaries affected. Variables come from YAML files (same trust level as code). `--set` overrides come from the CLI invocation (same trust level as the operator).

## Notes

2026-02-23 12:00 — Contract created. Depends on `origami-dsl-transformers`. This contract makes the circuit YAML self-sufficient by declaring all configuration and data flow inline.

2026-02-18 — Contract completed as part of DSL C1-C5 initiative. All phases executed, green gates passed across Origami, Asterisk, and Achilles.
