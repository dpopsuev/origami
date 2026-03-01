# Contract — origami-dsl-hooks

**Status:** complete  
**Goal:** Side effects (store writes, API calls, notifications) are declared as `after:` hooks on nodes in YAML, implemented as registered Go functions. Hooks don't affect routing or data flow.  
**Serves:** Origami DSL

## Contract rules

Global rules only, plus:

- **Part of a 5-contract series.** Contract 4 of 5. Depends on `origami-dsl-circuit-vars` (C3). Required by `origami-dsl-runner` (C5).
- **Hooks are side effects, not logic.** Hooks receive the validated artifact and can perform side effects (store writes, notifications). They do NOT affect routing (that's edges) or data flow (that's transformers). This separation is the Ansible `notify` / `handler` pattern.
- **Hooks are the Go escape hatch for domain logic.** When a domain needs to do something that can't be expressed in YAML (database writes, custom API calls), it registers a hook. This is the "high ceiling" in the Papert model.

## Context

- `origami-dsl-circuit-vars` — Predecessor. Provides unified context that hooks can read.
- `asterisk/internal/orchestrate/runner.go` — `ApplyStoreEffects` switch statement (5 step-specific side effects) that becomes registered hooks.
- Ansible `notify` / `handler` pattern — inspiration for the design.

## FSC artifacts

| Artifact | Target | Compartment |
|----------|--------|-------------|
| Hook pattern reference | `docs/` | domain |

## Execution strategy

Phase 1: Design `Hook` interface and `HookRegistry`. Phase 2: Add `After` field to `NodeDef` and integrate into Walk loop. Phase 3: Migrate Asterisk store effects to registered hooks. Phase 4: Validate, tune, validate.

## Coverage matrix

| Layer | Applies | Rationale |
|-------|---------|-----------|
| **Unit** | yes | Hook registration, invocation, error handling |
| **Integration** | yes | Walk loop invokes hooks after schema validation, before edge evaluation |
| **Contract** | yes | `Hook` interface compliance; `after:` YAML field accepted |
| **E2E** | yes | Asterisk stub calibration with hook-based side effects |
| **Concurrency** | no | Hooks are invoked sequentially per node |
| **Security** | no | Hooks are registered in Go code (same trust level as application) |

## Tasks

All tasks complete.

## Acceptance criteria

**Given** a circuit YAML with `after:` declarations on nodes,  
**When** a node completes and its artifact passes schema validation,  
**Then**:
- Each named hook in `after:` is invoked in order with the validated artifact
- Hook errors are logged (and optionally fail the walk)
- Hooks do NOT affect edge evaluation or data flow
- Asterisk store effects work as registered hooks
- `go build ./...` and `go test ./...` pass in Origami, Asterisk, and Achilles

## Security assessment

No new trust boundaries affected. Hooks are Go functions registered by the application at startup (same trust level as application code). Hook names come from YAML (same trust level as code).

## Notes

2026-02-23 12:00 — Contract created. Depends on `origami-dsl-circuit-vars`. This contract separates side effects from circuit logic, following the Ansible notify/handler pattern.

2026-02-18 — Contract completed as part of DSL C1-C5 initiative. All phases executed, green gates passed across Origami, Asterisk, and Achilles.
