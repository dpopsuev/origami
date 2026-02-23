# Contract — origami-dsl-runner

**Status:** complete  
**Goal:** `origami run pipeline.yaml --input data.json` works as a standalone CLI. The framework is a complete pipeline execution engine. Domain tools become thin wrappers.  
**Serves:** Origami DSL

## Contract rules

Global rules only, plus:

- **Part of a 5-contract series.** Contract 5 of 5. Depends on `origami-dsl-hooks` (C4). This is the capstone.
- **The low floor.** This contract delivers the Papert entry point: write a YAML pipeline, run `origami run`. No Go code needed for the common case.
- **Go API for embedding.** Domain tools that need custom setup (data gathering, hook registration) call `origami.Run()` from Go. The CLI is a thin wrapper around this API.

## Context

- `origami-dsl-hooks` — Predecessor. Provides the complete pipeline execution model: transformers + schemas + expressions + hooks.
- `origami-dsl-expression-engine` — Provides `when:` edge evaluation.
- `origami-dsl-transformers` — Provides built-in transformers (`llm`, `http`, `jq`, `file`).
- `origami-dsl-pipeline-vars` — Provides `vars:`, `input:`, prompt templates.
- `asterisk/cmd/asterisk/` — Current Asterisk CLI (~1500 lines) that becomes ~300 lines wrapping `origami.Run()`.
- `achilles/` — Second domain tool that validates the framework works for multiple domains.

## FSC artifacts

| Artifact | Target | Compartment |
|----------|--------|-------------|
| `origami run` CLI reference | `docs/` | domain |
| Go embedding API reference | `docs/` | domain |
| Updated Three CLIs pattern docs | `docs/` | domain |

## Execution strategy

Phase 1: Implement Go API `origami.Run()`. Phase 2: Implement `origami run` and `origami validate` CLI commands. Phase 3: Refactor Asterisk CLI to use `origami.Run()`. Phase 4: E2E validation across all repos. Phase 5: Validate, tune, validate.

## Coverage matrix

| Layer | Applies | Rationale |
|-------|---------|-----------|
| **Unit** | yes | Go API option parsing, YAML loading, variable override merging |
| **Integration** | yes | `origami.Run()` executes full pipeline with transformers + hooks + expressions |
| **Contract** | yes | CLI flag contract: `--input`, `--set`, positional pipeline path |
| **E2E** | yes | Asterisk stub calibration via `origami.Run()` produces identical metrics |
| **Concurrency** | no | Single pipeline execution per invocation |
| **Security** | yes | CLI accepts file paths and variable overrides from user input |

## Tasks

All tasks complete.

## Acceptance criteria

**Given** the Origami framework with all 5 DSL contracts complete,  
**When** a user runs `origami run pipeline.yaml --input data.json --set recall_hit=0.9`,  
**Then**:
- The pipeline loads, expressions compile, schemas validate, graph builds
- The walk executes: transformers process nodes, schemas validate artifacts, expressions route edges, hooks fire side effects
- `origami validate pipeline.yaml` catches errors without executing
- `origami.Run()` Go API works for embedding in domain tools
- Asterisk CLI is ~300 lines wrapping `origami.Run()` with RP data gathering and hook registration
- Achilles CLI works with the same `origami.Run()` API
- `go build ./...` and `go test ./...` pass in Origami, Asterisk, and Achilles

## Security assessment

| OWASP | Finding | Mitigation |
|-------|---------|------------|
| A01 Path Traversal | `--input` and positional pipeline path accept file paths from user. | Resolve paths relative to CWD. Reject absolute paths outside workspace. |
| A05 Misconfiguration | `--set` allows overriding any pipeline variable. A typo could disable safety thresholds. | Log all overrides at startup. `origami validate` warns about unknown variable names. |

## Notes

2026-02-23 12:00 — Contract created. Capstone of the Origami DSL initiative. Depends on all 4 prior contracts. Delivers the Papert low floor: `origami run pipeline.yaml`.

2026-02-18 — Contract completed as part of DSL C1-C5 initiative. All phases executed, green gates passed across Origami, Asterisk, and Achilles.
