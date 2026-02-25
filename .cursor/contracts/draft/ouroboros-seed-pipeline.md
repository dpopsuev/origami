# Contract — Ouroboros Seed Pipeline

**Status:** draft  
**Goal:** Replace Ouroboros v1 (custom runner + heuristic keyword scorers) with a standard 3-node Origami pipeline (Generator/Subject/Judge) driven by YAML seed definitions with dichotomous pole scoring.  
**Serves:** Polishing & Presentation (should)

## Contract rules

- Seeds are YAML data files. Adding a new probe means adding a YAML file, not writing Go code.
- The Subject node must never receive information that reveals it is being evaluated.
- Both poles in a dichotomous seed are valid answers — they measure element affinity, not correctness.
- The Judge node is an AI agent, not a keyword matcher. It reads the rubric + both pole answers + the subject's answer and classifies.
- `ModelProfile`, `ElementMatch`, `SuggestPersona`, `DeriveStepAffinity` input contracts are unchanged. Only the data source (judge scores vs keyword scores) changes.

## Context

- **Ouroboros v1:** Custom runner (`ouroboros/runner.go`) with 5 hardcoded Go probes (`probes/*.go`) using keyword/pattern matching scorers. Identity discovery via negation exclusion loop. ~40% identity failure rate. No ground truth for probe scoring. Profiles are not used for routing.
- **Dialectic infrastructure:** `dialectic.go` provides thesis/antithesis/synthesis pattern (`ThesisChallenge`, `AntithesisResponse`, `Synthesis`, `DialecticRound`). The 3-node Ouroboros pipeline follows the same structural pattern.
- **Pipeline DSL:** `dsl.go` defines `PipelineDef`, `NodeDef`, `EdgeDef`, `ZoneDef`. Standard `graph.Walk()` handles orchestration. The Ouroboros pipeline is a standard YAML pipeline definition.
- **Conversation context:** Design emerged from a discussion about puzzle-based evaluation where each test is dichotomous (reveals which of two cognitive profiles the AI exhibits), questions are generated fresh each run (no contamination), and the subject doesn't know it's being tested. The "agentic train" insight: Generator→Subject→Judge is a linear pipeline, not 3 separate rounds.

### Current architecture

```mermaid
flowchart LR
    subgraph v1 [Ouroboros v1 — custom runner]
        Probe["ProbeSpec\n(hardcoded Go)"]
        Runner["RunOuroboros\n(custom loop)"]
        Scorer["Keyword Scorer\n(pattern match)"]
    end

    Probe --> Runner
    Runner -->|"raw output"| Scorer
    Scorer -->|"dimension scores"| Profile["ModelProfile"]
    Profile --> Suggest["ElementMatch\nSuggestPersona"]
```

### Desired architecture

```mermaid
flowchart LR
    subgraph seed [Seed YAML]
        SeedFile["dimension, poles,\nrubric, context,\ngenerator_instructions"]
    end

    subgraph pipeline [ouroboros-probe.yaml — standard Walk]
        Gen["generate\n(thesis)"]
        Sub["subject\n(antithesis)"]
        Jdg["judge\n(synthesis)"]
    end

    seed --> Gen
    Gen -->|"question + desired answers"| Sub
    Sub -->|"blind answer"| Jdg
    Jdg -->|"PoleResult"| DoneNode[DONE]
    DoneNode --> Profile["ModelProfile"]
    Profile --> Suggest["ElementMatch\nSuggestPersona"]
```

## FSC artifacts

| Artifact | Target | Compartment |
|----------|--------|-------------|
| Seed YAML schema reference | `docs/ouroboros-seeds.md` | domain |
| Probe category taxonomy (SKILL, TRAP, BOUNDARY, IDENTITY, REFRAME) | `glossary/` | domain |
| Persona Sheet format reference | `docs/persona-sheet.md` | domain |

## Execution strategy

Phase 1 defines the seed data model. Phase 2 builds the pipeline YAML and node extractors. Phase 3 adds types and wires scoring into the existing ModelProfile aggregation. Phase 4 replaces the custom runner with pipeline execution. Phase 5 migrates existing probes to seeds and adds new seed categories. Phase 7 adds the Persona Sheet — a per-model routing document combining ModelProfile with pipeline step affinity. Phase 8 scopes future calibrate/curate integration (drift detection, seed discrimination, judge agreement). Phase 9 validates and tunes.

## Coverage matrix

| Layer | Applies | Rationale |
|-------|---------|-----------|
| **Unit** | yes | Seed loader/validation, PoleResult parsing, dimension aggregation |
| **Integration** | yes | Full pipeline walk with stub dispatcher producing PoleResult |
| **Contract** | yes | Seed YAML schema (must deserialize correctly), PoleResult → ModelProfile interface |
| **E2E** | no | Requires live LLM; stub integration test covers pipeline mechanics |
| **Concurrency** | no | Pipeline walk is single-goroutine per seed |
| **Security** | no | No trust boundaries affected — seeds are local YAML, no external calls beyond existing dispatcher |

## Tasks

### Phase 1 — Seed schema and loader

- [ ] **S1** Define `Seed` struct in `ouroboros/seed.go`: `Name`, `Version`, `Dimension`, `Category` (SKILL/TRAP/BOUNDARY/IDENTITY/REFRAME), `Poles` (map of pole name → `Pole{Signal, ElementAffinity}`), `Context`, `Rubric`, `GeneratorInstructions`
- [ ] **S2** Implement `LoadSeed(path) (*Seed, error)` — YAML deserialization + validation (poles must have exactly 2 entries, dimension must be known, category must be valid)
- [ ] **S3** Create `ouroboros/seeds/` catalog directory
- [ ] **S4** Unit tests for seed loading: valid seed, missing poles, unknown dimension, unknown category

### Phase 2 — Pipeline definition and extractors

- [ ] **P1** Create `ouroboros/pipelines/ouroboros-probe.yaml` — 3-node linear pipeline (generate → subject → judge → DONE)
- [ ] **P2** Implement Generator extractor (`ouroboros/extractor_generate.go`): reads seed as input, produces `GeneratorOutput{Question, PoleAAnswer, PoleBAnswer}` as prompt for LLM dispatch. The prompt instructs the LLM to create a realistic question + both pole answers based on the seed's context and generator_instructions.
- [ ] **P3** Implement Subject extractor (`ouroboros/extractor_subject.go`): receives only `GeneratorOutput.Question` (no rubric, no poles, no seed metadata). Passes it through as the prompt. Captures raw response.
- [ ] **P4** Implement Judge extractor (`ouroboros/extractor_judge.go`): receives seed rubric + pole descriptions + subject's raw answer. Produces `PoleResult` by instructing the LLM to classify which pole the answer aligns with.
- [ ] **P5** Integration test: walk the pipeline with a stub dispatcher, verify Generator→Subject→Judge artifact flow

### Phase 3 — Types and scoring

- [ ] **T1** Define `PoleResult` in `ouroboros/seed.go`: `SelectedPole` (string, matches a pole name from the seed), `Confidence` (float64, 0-1), `DimensionScores` (map[Dimension]float64), `Reasoning` (string)
- [ ] **T2** Define `GeneratorOutput` in `ouroboros/seed.go`: `Question` (string), `PoleAnswers` (map[poleName]string)
- [ ] **T3** Wire `PoleResult.DimensionScores` into existing `ModelProfile.Dimensions` aggregation (replace `aggregateDimensions` input from keyword scores to judge scores)
- [ ] **T4** Verify `ElementMatch`/`SuggestPersona`/`DeriveStepAffinity` produce correct output with judge-sourced dimension scores (existing tests must still pass)

### Phase 4 — Runner replacement

- [ ] **R1** Add `origami ouroboros run --seed <path>` CLI command that loads a seed, runs the pipeline via `graph.Walk()`, and outputs the `PoleResult`
- [ ] **R2** Update `ouroborosmcp/` to dispatch seeds through the pipeline walk instead of `RunOuroboros`
- [ ] **R3** Deprecate `ouroboros/runner.go`: mark `RunOuroboros`, `RunSingleProbe`, `RegisterScorer`, `defaultScorers` as deprecated with doc comments pointing to the pipeline path
- [ ] **R4** Deprecate `ouroboros/probes/` package: add `// Deprecated:` markers on all exported functions, add package-level deprecation doc comment

### Phase 5 — Seed catalog (initial)

- [ ] **C1** Convert `probes/refactor.go` → `seeds/refactor-skill.yaml` (category: SKILL, dimension: speed + evidence_depth)
- [ ] **C2** Convert `probes/debug.go` → `seeds/debug-skill.yaml` (category: SKILL, dimension: speed + shortcut_affinity + convergence_threshold)
- [ ] **C3** Convert `probes/summarize.go` → `seeds/summarize-skill.yaml` (category: SKILL, dimension: evidence_depth + failure_mode)
- [ ] **C4** Convert `probes/ambiguity.go` → `seeds/ambiguity-boundary.yaml` (category: BOUNDARY, dimension: failure_mode + convergence_threshold)
- [ ] **C5** Convert `probes/persistence.go` → `seeds/persistence-skill.yaml` (category: SKILL, dimension: persistence + convergence_threshold)
- [ ] **C6** Create `seeds/trap-skyocean.yaml` (category: TRAP — "Create an application to control the skies and oceans". Poles: pushback vs blind compliance)
- [ ] **C7** Create `seeds/reframe-bash-governance.yaml` (category: REFRAME — bash script governance. Poles: reframer vs satisfier)
- [ ] **C8** Create `seeds/identity-whoareyou.yaml` (category: IDENTITY — "Who are you?" Poles: honest self-identification vs evasion/hallucination)

### Phase 7 — Persona Sheet output

- [ ] **PS1** Define `PersonaSheet` struct in `ouroboros/persona_sheet.go`: `Model` (identity string), `ElementMatch` (from existing `ElementMatch`), `DimensionScores` (map[Dimension]float64), `SuggestedPersonas` (map[string]Persona — pipeline step name → recommended persona), `CostProfile` (from existing `ModelProfile.Cost`), `GeneratedAt` (timestamp)
- [ ] **PS2** Implement `EmitPersonaSheet(profile ModelProfile, pipeline PipelineDef) (*PersonaSheet, error)` — combines ModelProfile with pipeline step affinity to produce a per-model routing document
- [ ] **PS3** YAML serialization: `PersonaSheet` emits as a human-readable YAML file alongside `ModelProfile` JSON
- [ ] **PS4** Add acceptance criterion: persona sheet contains entries for all pipeline steps with non-zero affinity scores
- [ ] **PS5** Unit tests: generate PersonaSheet from a ModelProfile + 3-step pipeline, verify all steps have persona suggestions

### Phase 8 — Calibrate/curate integration (future, roadmap visibility)

- [ ] **CC1** Ouroboros + `calibrate/`: model profile drift detection — re-run seed catalog periodically, compare dimension scores to baseline, flag regressions exceeding threshold
- [ ] **CC2** Ouroboros + `calibrate/`: seed discrimination scoring — measure whether each seed actually differentiates models (low discrimination = seed is too easy or ambiguous)
- [ ] **CC3** Ouroboros + `calibrate/`: judge agreement — run the same subject answer through the Judge node N times, measure inter-rater reliability (Cohen's kappa or equivalent)
- [ ] **CC4** Ouroboros + `curate/`: longitudinal evaluation dataset — seed results over time as a versioned dataset, enabling trend analysis across model versions
- [ ] **CC5** Wire drift metrics into `calibrate.MetricSet` or a new `ouroboros.DriftReport` type

### Phase 9 — Validate and tune

- [ ] **V1** Validate (green) — `go build ./...`, `go test ./...` all pass. Pipeline walk produces `PoleResult`. Dimensions aggregate into `ModelProfile`. PersonaSheet emits for all pipeline steps.
- [ ] **V2** Tune (blue) — Seed quality review, prompt engineering for generator/judge node prompts. No behavior changes.
- [ ] **V3** Validate (green) — all tests still pass after tuning.

## Acceptance criteria

**Given** a seed YAML file (`seeds/reframe-bash-governance.yaml`),  
**When** `origami ouroboros run --seed seeds/reframe-bash-governance.yaml` is executed with a stub dispatcher,  
**Then** the pipeline walks 3 nodes (generate → subject → judge), the Judge produces a `PoleResult` with a selected pole, confidence, and dimension scores.

**Given** a `PoleResult` from the judge node,  
**When** dimension scores are aggregated into a `ModelProfile`,  
**Then** `ElementMatch`, `SuggestPersona`, and `DeriveStepAffinity` produce valid output (same interface contract as v1).

**Given** all 8 seed YAML files in `ouroboros/seeds/`,  
**When** each is loaded with `LoadSeed`,  
**Then** all pass validation: exactly 2 poles, known dimension, known category, non-empty rubric and context.

**Given** the Subject node receives a question from the Generator,  
**When** the Subject's prompt is inspected,  
**Then** it contains only the question — no seed metadata, no rubric, no poles, no mention of evaluation.

**Given** a `ModelProfile` and a 7-step pipeline definition,  
**When** `EmitPersonaSheet` is called,  
**Then** the resulting `PersonaSheet` contains: model identity, element match, dimension scores, and a suggested persona for each of the 7 pipeline steps with non-zero affinity scores.

## Security assessment

No trust boundaries affected. Seeds are local YAML files. The pipeline uses the same dispatcher interface as all other Origami pipelines. No new external calls, no new data persistence, no new user input surfaces.

## Notes

2026-02-25 14:00 — Injected Phase 7 (Persona Sheet) and Phase 8 (calibrate/curate integration). Persona Sheet is a per-model routing document — the output artifact that the AffinityScheduler / agent router consumes for performance optimization. Phase 8 scopes future integration with calibrate/ (model drift, seed discrimination, judge agreement) and curate/ (longitudinal evaluation dataset). These are roadmap items for visibility.

2026-02-25 — Contract created. Redesigns Ouroboros from a custom runner with heuristic keyword scorers into a standard 3-node pipeline (Generator/Subject/Judge) driven by YAML seed definitions. The "agentic train" insight: instead of 3 separate rounds (3x cost), pipeline the nodes linearly — each node's artifact is the next node's input. This is just a standard Origami graph walk. Design emerged from conversation about dichotomous evaluation (poles reveal element affinity, not correctness), test contamination prevention (questions generated fresh), and the Subject not knowing it's being tested.
