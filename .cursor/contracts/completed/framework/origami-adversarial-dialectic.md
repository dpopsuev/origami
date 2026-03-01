# Contract — origami-adversarial-dialectic

**Status:** complete  
**Goal:** All "Court" terminology is replaced with "Dialectic" terminology in code and FSC, codifying the thesis-antithesis-synthesis model for adversarial graph execution.  
**Serves:** Framework Maturity (current goal)

## Contract rules

Global rules only, plus:

- **Rename only, no behavior change.** The adversarial circuit logic is unchanged. This is a terminology migration to align code with the dialectic model described in `strategy/origami-vision.mdc`.
- **All references.** Code, tests, FSC docs, glossary, comments, struct names, function names, file names.

## Context

- `strategy/origami-vision.mdc` — Defines the dialectic model and terminology mapping.
- `glossary/glossary.mdc` — Already updated with dialectic terms; old "Shadow Court" entry replaced.
- `court.go` — Current implementation of adversarial circuit logic.
- `court_test.go` — Tests for adversarial circuit.
- `identity.go` — May reference Court-related types.
- `completed/framework/agentic-framework-III.3-shadow.md` — Historical contract that introduced the Court.
- `completed/framework/defect-court.md` — Historical contract.

### Terminology mapping

| Old | New | Type |
|---|---|---|
| `court.go` | `dialectic.go` | File |
| `court_test.go` | `dialectic_test.go` | File |
| `CourtConfig` | `DialecticConfig` | Struct |
| `ShouldActivate` | `NeedsAntithesis` | Method |
| `MaxHandoffs` | `MaxTurns` | Field |
| `MaxRemands` | `MaxNegations` | Field |
| `ActivationThreshold` | `ContradictionThreshold` | Field (if exists) |
| "Shadow Court" | "Adversarial Dialectic" | Prose |
| "Prosecution" / "Defense" | "Thesis-holder" / "Antithesis-holder" | Prose |
| "Verdict" | "Synthesis" | Prose |
| "Mistrial" | "Unresolved contradiction" | Prose |

## FSC artifacts

| Artifact | Target | Compartment |
|----------|--------|-------------|
| Updated glossary terms | `glossary/` | domain |

## Execution strategy

Phase 1: Rename Go source files (`court.go` -> `dialectic.go`, etc.). Phase 2: Rename structs, methods, and fields. Phase 3: Update all prose references in code comments and FSC. Phase 4: Validate, tune, validate.

## Coverage matrix

| Layer | Applies | Rationale |
|-------|---------|-----------|
| **Unit** | yes | All court/dialectic tests must pass with new names |
| **Integration** | no | No cross-boundary changes |
| **Contract** | yes | Exported type names change — verify consumers compile |
| **E2E** | no | No behavior change |
| **Concurrency** | no | No concurrency changes |
| **Security** | no | No trust boundaries affected |

## Tasks

- [x] Rename `court.go` -> `dialectic.go`, `court_test.go` -> `dialectic_test.go`
- [x] Rename `CourtConfig` -> `DialecticConfig`, `ShouldActivate` -> `NeedsAntithesis`, `MaxHandoffs` -> `MaxTurns`, `MaxRemands` -> `MaxNegations`
- [x] Update all code comments referencing "court", "verdict", "prosecution", "defense", "mistrial"
- [x] Update consumer code (Asterisk, Achilles) if they reference Court types
- [x] Verify glossary already updated (done in prior housekeeping)
- [x] Validate (green) — `go build ./...` and `go test ./...` pass in Origami, Asterisk, Achilles
- [x] Tune (blue) — review all renames for consistency
- [x] Validate (green) — all tests still pass after tuning

## Acceptance criteria

**Given** the Origami codebase and FSC,  
**When** this contract is complete,  
**Then**:
- Zero occurrences of `CourtConfig`, `ShouldActivate`, `MaxHandoffs`, `MaxRemands` in Go source
- Zero occurrences of "Shadow Court" in FSC files (except historical completed contracts)
- `DialecticConfig`, `NeedsAntithesis`, `MaxTurns`, `MaxNegations` are the exported names
- `dialectic.go` and `dialectic_test.go` exist; `court.go` and `court_test.go` do not
- `go build ./...` and `go test ./...` pass in Origami, Asterisk, and Achilles

## Security assessment

No trust boundaries affected. Pure rename operation.

## Notes

2026-02-18 — Contract created. Gate contract for Framework Maturity goal. Closes gap #6 from the maturity assessment.
