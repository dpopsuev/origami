# Contract — thesis-antithesis-rename

**Status:** complete  
**Goal:** Rename Light/Shadow alignment terminology to Thesis/Antithesis across Origami and Asterisk.  
**Serves:** API Stabilization

## Contract rules

- Origami rename lands first; Asterisk follows.
- Persona names (Herald, Seeker, Challenger, Abyss, etc.) and mythological names (Cadai, Cytharai) are unchanged.
- Artifact types (ThesisChallenge, AntithesisResponse, SynthesisDecision) are already correct — no changes needed.
- Serialized alignment values change from `"light"`/`"shadow"` to `"thesis"`/`"antithesis"` — acceptable breaking change per API Stabilization goal (no persisted stores, no external consumers).

## Context

- [Electronic Circuit Case Study](../../docs/case-studies/electronic-circuit-theory.md) — Op-Amp analogy revealed that persona alignment names should mirror the dialectic structure.
- [Glossary](../../glossary/glossary.mdc) — Already defines Thesis/Antithesis as dialectic concepts; Light/Shadow as alignment terms. Post-rename these converge.

### Current architecture

The Adversarial Dialectic names its artifacts `ThesisChallenge`, `AntithesisResponse`, `SynthesisDecision`, but the personas that produce them use `AlignmentLight` / `AlignmentShadow`. This creates a terminology split.

### Desired architecture

Alignment names match dialectic structure: `AlignmentThesis` / `AlignmentAntithesis`. The philosophical foundation (Hegel), the artifact types, and the persona alignments all speak the same language.

## FSC artifacts

| Artifact | Target | Compartment |
|----------|--------|-------------|
| Updated glossary entries | `glossary/` | domain |

## Execution strategy

1. Rename Go identifiers in Origami (identity.go, persona.go, mask.go, curate/walker.go, examples/playground/main.go).
2. Rename test names and assertions (persona_test.go, mask_test.go).
3. Update Go comments (persona.go, dialectic.go, evidence_gap.go, mask.go, lsp/hover.go).
4. Update YAML comments and all documentation files.
5. Update glossary entries.
6. Build + test Origami.
7. Update Asterisk Go code and documentation.
8. Build + test Asterisk.

## Coverage matrix

| Layer | Applies | Rationale |
|-------|---------|-----------|
| **Unit** | yes | persona_test.go, mask_test.go — renamed functions and assertions |
| **Integration** | yes | `go build ./...` on both repos with updated dependency |
| **Contract** | no | No API schema changes beyond the alignment string values |
| **E2E** | no | No pipeline behavior change, only names |
| **Concurrency** | no | No shared state affected |
| **Security** | no | No trust boundaries affected |

## Tasks

- [x] Rename Go identifiers in Origami (identity.go, persona.go, mask.go, curate/walker.go, examples/playground/main.go)
- [x] Rename test names and assertions (persona_test.go, mask_test.go)
- [x] Update Go comments (persona.go, dialectic.go, evidence_gap.go, mask.go, lsp/hover.go)
- [x] Update YAML comments (2 files) and all documentation (11 files) in Origami
- [x] Update glossary entries
- [x] Build + test Origami green
- [x] Update Asterisk Go code (3 files) and documentation (4 files)
- [x] Build + test Asterisk green
- [x] Validate (green) — all tests pass, acceptance criteria met.
- [x] Tune (blue) — no further refactoring needed; rename was clean.
- [x] Validate (green) — all tests still pass after tuning.

## Acceptance criteria

- `go build ./...` passes in both Origami and Asterisk.
- `go test ./...` passes in both Origami and Asterisk.
- Zero occurrences of `AlignmentLight`, `AlignmentShadow`, `LightPersonas`, `ShadowPersonas`, `DefaultLightMasks` in Go code.
- Zero occurrences of "Light path", "Shadow path", "Light pipeline", "Shadow pipeline" in Go comments.
- Glossary entries updated to reflect Thesis/Antithesis as alignment terms.
- Achilles requires zero changes (verified by search).

## Security assessment

No trust boundaries affected.

## Notes

2026-03-01 — Contract created from electronic circuit case study insight. Op-Amp analogy (non-inverting amplifier topology) revealed that thesis/antithesis alignment names complete the Hegelian dialectic terminology chain already present in artifact types.
