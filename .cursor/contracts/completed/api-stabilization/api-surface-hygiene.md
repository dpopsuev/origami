# Contract — api-surface-hygiene

**Status:** complete  
**Goal:** Execute all P1 quick fixes and P2 test gaps from the public API surface audit, closing "Public API surface is stable" and "Test coverage on public interfaces is solid".  
**Serves:** API Stabilization (finish-line)

## Contract rules

Global rules only, plus:

- **No behavioral changes.** P1 fixes remove dead code, fix comments, and rename test symbols. No runtime behavior changes.
- **Tests exercise the public API.** P2 tests call exported functions/constructors directly. No testing internals via reflection or unexported fields.
- **One commit per group.** P1 fixes in one commit, P2 test gaps in a second. Build + test green before each commit.

## Context

- `contracts/active/public-api-surface-audit.md` — the audit that produced these findings
- `current-goal.mdc` — two open checklist items this contract closes

### Source findings

| ID | Finding | File | Lines |
|----|---------|------|-------|
| P1-1 | Dead export `ResolveFQCN` (zero callers, name collision with `fold/ModuleRegistry.ResolveFQCN`) | `component.go` | 124-127 |
| P1-2 | Stale comment on `runExprProgram` says "Exported for testing" but function is unexported | `expression_edge.go` | 83-84 |
| P1-3 | Stale "Adapter" terminology in test (`TestBatchWalk_PerCaseAdapters`, `hookAdapter`, `noopHookAdapter`) | `batch_walk_test.go` | 116, 138, 149 |
| P1-4 | 14 lint rule types lack godoc | `lint/rules_structural.go` | 47-405 |
| P2-1 | `ExecHook`, `NewExecHook` — zero tests | `components/sqlite/hook.go` | 25, 30-33 |
| P2-2 | `CoreComponent`, `WithCoreBaseDir`, `TemplateParamsTransformer` — zero tests | `transformers/core_component.go`, `template_params.go` | 11, 24 |
| P2-3 | `HasOTelEndpoint` — zero tests | `observability/defaults.go` | 29-30 |
| P2-4 | `StaticDispatcher`, `NewStaticDispatcher` — zero tests | `dispatch/static.go` | 25, 32 |
| P2-5 | `NewStdioStream`, `ServeStream` — zero tests | `lsp/stdio.go` | 17-18, 22-23 |

## FSC artifacts

Code only — no FSC artifacts.

## Execution strategy

Two phases, sequential. P1 first (no new tests, just cleanup), then P2 (new test files). Build + lint + test after each phase.

### Phase 1 — Quick fixes

1. Delete `ResolveFQCN` from `component.go` and `TestResolveFQCN` from `component_test.go`.
2. Fix the misleading comment on `runExprProgram` in `expression_edge.go`.
3. Rename `TestBatchWalk_PerCaseAdapters` -> `TestBatchWalk_PerCaseComponents`, `hookAdapter` -> `hookComponent`, `noopHookAdapter` -> `noopHookComponent` in `batch_walk_test.go`.
4. Add one-line godoc to each of the 14 exported rule types in `lint/rules_structural.go`.

### Phase 2 — Test gaps

5. Add `components/sqlite/hook_test.go` — test `NewExecHook` construction and `ExecHook` execution.
6. Add `transformers/core_component_test.go` and/or `template_params_test.go` — test `CoreComponent` registration and `TemplateParamsTransformer` behavior.
7. Add `HasOTelEndpoint` test to `observability/observability_test.go` — verify env var detection.
8. Add `dispatch/static_test.go` — test `NewStaticDispatcher` construction and dispatch behavior.
9. Add `lsp/stdio_test.go` — test `NewStdioStream` and `ServeStream` with pipe-based I/O.

## Coverage matrix

| Layer | Applies | Rationale |
|-------|---------|-----------|
| **Unit** | yes | P2 adds unit tests for 9 previously untested exports across 5 packages |
| **Integration** | no | All tests exercise single-package exports; no cross-boundary interactions |
| **Contract** | no | No API schema changes |
| **E2E** | no | No circuit-level changes |
| **Concurrency** | no | No shared state introduced |
| **Security** | no | No trust boundaries affected |

## Tasks

- [x] **P1-1** Delete `ResolveFQCN` from `component.go` and `TestResolveFQCN` from `component_test.go`
- [x] **P1-2** Fix stale comment on `runExprProgram` in `expression_edge.go`
- [x] **P1-3** Rename `Adapter` -> `Component` in `batch_walk_test.go` (test name + variables)
- [x] **P1-4** Add godoc to 14 lint rule types in `lint/rules_structural.go`
- [x] **P2-1** Add `components/sqlite/hook_test.go` for `ExecHook` / `NewExecHook`
- [x] **P2-2** Add tests for `CoreComponent` / `WithCoreBaseDir` / `TemplateParamsTransformer` in `transformers/`
- [x] **P2-3** Add test for `HasOTelEndpoint` in `observability/`
- [x] **P2-4** Add `dispatch/static_test.go` for `StaticDispatcher`
- [x] **P2-5** Add `lsp/stdio_test.go` for `NewStdioStream` / `ServeStream`
- [x] Validate (green) — `go build ./...` and `go test -race ./...` pass, all 37 packages green.
- [x] Tune (blue) — no further tuning needed; changes are minimal and clean.
- [x] Validate (green) — all tests pass after tuning.

## Acceptance criteria

**Given** `ResolveFQCN` is deleted from `component.go`,  
**When** `go build ./...` runs,  
**Then** the build succeeds with zero references to the deleted function.

**Given** all 14 lint rule types in `lint/rules_structural.go`,  
**When** inspected,  
**Then** each has a `// TypeName ...` godoc comment.

**Given** `batch_walk_test.go`,  
**When** inspected,  
**Then** no symbol contains the substring "Adapter" (all renamed to "Component").

**Given** P2 test files,  
**When** `go test ./components/sqlite/ ./transformers/ ./observability/ ./dispatch/ ./lsp/` runs,  
**Then** the new tests pass and cover the previously untested exports.

**Given** the full test suite,  
**When** `go test -race ./...` runs,  
**Then** all tests pass with zero race conditions.

## Security assessment

No trust boundaries affected. P1 removes dead code and fixes comments. P2 adds test files. No runtime behavior changes.

## Notes

2026-03-01 — Contract created from public-api-surface-audit findings. Corrected audit errors: `runExprProgram` is already unexported (comment fix only), `StdinDispatcher` already has 4 tests in `stdin_template_test.go` (dropped from P2).

2026-03-02 — Reassessed all 9 findings against current codebase; all still valid. Executed both phases. P1: deleted `ResolveFQCN` (+ removed unused `strings` import), fixed `runExprProgram` comment, renamed 4 "Adapter" symbols in `batch_walk_test.go`, added godoc to 14 lint rule types. P2: added 5 test files covering `ExecHook`, `CoreComponent`, `TemplateParamsTransformer`, `HasOTelEndpoint`, `StaticDispatcher`, `NewStdioStream`, `ServeStream`. Full suite: 37 packages, zero failures, zero races.
