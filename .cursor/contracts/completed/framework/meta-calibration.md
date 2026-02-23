# Contract — Meta-calibration (Ouroboros)

**Status:** complete  
**Goal:** Task-based behavioral assessment of LLM models, producing empirical ModelProfiles that replace hardcoded persona traits with measured data. The recursive tuning cycle is named **Ouroboros**.  
**Serves:** Architecture evolution (vision — Framework Tome IV)

## Contract rules

- Zero imports from Asterisk domain packages (`calibrate`, `orchestrate`, `origami`). This is a framework-level concern.
- Probes must be task-based (observe behavior), never self-report (ask the model about itself).
- Historical profiles are append-only. Never delete or overwrite a past calibration result.
- All scores are continuous (0.0-1.0), not binary. Normalization is relative (percentiles across registry).

## Context

### The MBTI analogy

MBTI measures personality via 4 dichotomies (E/I, S/N, T/F, J/P) using self-report questionnaires. Its weakness: self-reporting is subject to the Barnum effect, confirmation bias, and social desirability. The test has poor test-retest reliability (39-76% of people get a different type on retest).

Agent meta-calibration inverts this:

| MBTI | Agent Meta-calibration |
|------|----------------------|
| Self-report questionnaire | Task-based behavioral probes |
| "Are you organized?" | Give messy code, observe if agent restructures |
| 4 dichotomies, 16 types | 6 behavioral dimensions, 6 Elements |
| Static (one test, one time) | Re-run when models change (relative, not absolute) |
| Binary (E vs I) | Continuous scores (0.0-1.0 per dimension) |
| No predictive power | Empirically correlated to domain calibration performance |

Key design choice: the agent does NOT know which dimension is being measured. It just does a task. We observe the behavioral signals in the output.

### Current architecture

Persona traits in `internal/framework/persona.go` are hardcoded by human judgment:
```go
StepAffinity: map[string]float64{
    "recall": 0.9, "triage": 0.8,    // <-- human guesses
    "resolve": 0.3, "investigate": 0.2,
}
```

`IronFromEarth(accuracy)` in `internal/framework/element.go` already derives element traits from calibration data. Meta-calibration generalizes this pattern to all elements and all models.

`KnownModels` in `internal/framework/known_models.go` is a static registry. Meta-calibration adds a dynamic profile layer on top.

### Desired architecture

```
internal/framework/metacal/
├── types.go          # ModelProfile, ProbeSpec, ProbeResult, Dimension
├── battery.go        # Battery definition, probe registry
├── probes/           # Individual probe implementations
│   ├── refactor.go   # Structure vs flexibility probe
│   ├── debug.go      # Speed, shortcut, convergence probe
│   └── summarize.go  # Evidence depth, failure mode probe
├── runner.go         # RunBattery, per-probe execution
├── normalize.go      # Percentile normalization across profiles
├── store.go          # ProfileStore (append-only JSON)
├── suggest.go        # SuggestPersona, ElementMatch
└── *_test.go
```

## FSC artifacts

| Artifact | Target | Compartment |
|----------|--------|-------------|
| Meta-calibration design reference | `docs/meta-calibration.md` | domain |
| Behavioral dimension glossary | `glossary/` | domain |

## Execution strategy

Build types first (Phase 1), then implement probes (Phase 2), then the runner and normalization (Phase 3), then wire to personas (Phase 4). Each phase is independently testable. The package has zero domain imports at all times.

## Tasks

### Phase 1 — Types and probe specs

- [x] Define `ModelProfile` struct (Model, BatteryVersion, Timestamp, Dimensions, ElementMatch, ElementScores, SuggestedPersonas, CostProfile, RawResults)
- [x] Define `ProbeSpec` struct (ID, Name, Description, Dimensions it measures, Input data, ExpectedBehaviors)
- [x] Define `ProbeResult` struct (ProbeID, RawOutput, DimensionScores, Elapsed, TokensUsed)
- [x] Define `Dimension` enum (Speed, Persistence, ConvergenceThreshold, ShortcutAffinity, EvidenceDepth, FailureMode)
- [x] Define `ProfileStore` interface (Save, Load, List, History for a model)
- [x] Implement `FileProfileStore` (JSON in `metacal/profiles/`, append-only)
- [x] Unit tests for all types and store

### Phase 2 — Initial probes (3-5)

- [x] **Refactoring probe** — Input: valid but messy code (no naming, monolithic function, no comments). Measures: structure (Earth/Diamond) vs speed (Fire/Lightning). Scorer: count renames, function splits, comments added, structural changes.
- [x] **Debugging probe** — Input: log output with one red herring and one subtle root cause. Measures: speed, shortcut affinity, convergence threshold. Scorer: which cause identified first, how many hypotheses explored, time to convergence.
- [x] **Summarization probe** — Input: large PR diff with mixed changes (feature + refactor + fix). Measures: evidence depth, failure mode. Scorer: how many distinct changes identified, accuracy of categorization, verbosity vs precision.
- [x] **Ambiguity probe** — Input: contradictory requirements. Measures: failure mode, convergence threshold. Scorer: does agent ask for clarification, pick one, or attempt both? Quality of reasoning about the contradiction.
- [x] **Persistence probe** — Input: task requiring backtracking (first approach fails). Measures: persistence (MaxLoops), convergence threshold. Scorer: does agent retry with different approach or give up?
- [x] Each probe: deterministic input, automated scorer, dimension score output
- [x] Unit tests for each probe scorer

### Phase 3 — Runner and normalization

- [x] Implement `RunOuroboros(ctx, model, dispatcher, battery) ModelProfile`
- [x] Implement `RunSingleProbe(ctx, dispatcher, spec) ProbeResult`
- [x] Implement percentile normalization: recompute all dimension scores as percentiles across stored profiles
- [x] Implement staleness detection: flag profiles where battery version changed, model version updated, or TTL exceeded
- [ ] Implement model popularity tracking: provider availability, deprecation status
- [x] Historical evolution view: compare same model across versions
- [x] Integration tests with stub dispatcher

### Phase 4 — Wire to personas

- [x] `ElementMatch(profile ModelProfile) Element` — map dimension scores to best-fit element
- [x] `ElementScores(profile ModelProfile) map[Element]float64` — affinity score to each element
- [x] `SuggestPersona(profile ModelProfile) []string` — recommend persona assignment
- [x] `DeriveStepAffinity(profile ModelProfile) map[string]float64` — replace hardcoded values
- [x] `IronFromProfile(profile ModelProfile) ElementTraits` — Ouroboros alternative to IronFromEarth
- [x] Unit tests for mapping and suggestion logic

### Phase 5 — Validate and tune

- [x] Validate (green) — all tests pass, zero domain imports, store round-trip works
- [x] Tune (blue) — refactor for quality, review dimension names, ensure probe scoring is deterministic
- [x] Validate (green) — all tests still pass after tuning

## Acceptance criteria

**Given** a registered model in `KnownModels`,  
**When** `RunBattery` is executed with the standard probe battery,  
**Then** a `ModelProfile` is produced with:
- All 6 dimensions scored (0.0-1.0)
- `ElementMatch` assigned to one of the 6 core elements
- `CostProfile` measured (tokens, latency)
- Profile persisted to append-only store
- Percentile normalization applied relative to all stored profiles

**Given** two models with different behavioral characteristics,  
**When** both are meta-calibrated,  
**Then** their `ElementMatch` values differ appropriately (e.g. a fast model maps to Fire/Lightning, a thorough model maps to Water/Earth).

**Given** a `ModelProfile`,  
**When** `SuggestPersona` is called,  
**Then** at least one persona is suggested with `StepAffinity` values derived from measured dimensions, not hardcoded.

## Security assessment

No trust boundaries affected. Meta-calibration probes operate on synthetic inputs only. No external API calls, no user data, no secrets.

## Notes

2026-02-21 12:00 — Contract created based on discussion about MBTI-inspired agent assessment. Key design decision: task-based behavioral probes, not self-report. Continuous scores, not binary types. Historical append-only storage for evolution tracking.

2026-02-21 21:30 — Pipeline named **Ouroboros** (recursive tuning cycle). All 5 phases implemented. Package: `pkg/framework/metacal/` + `probes/` subpackage. 5 probes (refactor, debug, summarize, ambiguity, persistence), `RunOuroboros` runner, percentile normalization, `ElementMatch`/`SuggestPersona`/`DeriveStepAffinity`/`IronFromProfile`. 56 tests across metacal + probes, all green. Zero domain imports. `Dispatcher` func type replaces domain `ModelAdapter`. One task deferred: model popularity tracking (future enhancement). Registered scorer pattern (`RegisterScorer`) allows probes/ package to wire scorers without circular imports.
