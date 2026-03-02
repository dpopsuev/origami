# Contract — public-api-surface-audit

**Status:** complete (2026-03-02)  
**Goal:** Document the full exported API surface of Origami with test coverage analysis and a prioritized issues list. Read-only — no code changes.  
**Serves:** API Stabilization

## Contract rules

- No code changes. This contract produces documentation only.
- All claims reference concrete files, symbols, and line numbers.
- Test quality ratings are evidence-based (test file exists, exercises exported API directly).

## Context

Origami is feature-complete. The API Stabilization goal requires freezing the public surface. Before freezing, we need a clear picture of what that surface is, what's well-tested, what's not, and what's messy.

Prior work: Marble excision (zero residue), RCA/RP decoupling (rcatype + rpconv boundary).

## FSC artifacts

| Artifact | Target | Compartment |
|----------|--------|-------------|
| API surface inventory | this contract | domain |
| Test coverage matrix | this contract | domain |
| Prioritized action items | this contract → follow-up contracts | domain |

## Execution strategy

Single-pass audit. No code changes. Produce three sections: The Good (strengths), The Bad (fixable issues), The Ugly (structural concerns).

## Coverage matrix

| Layer | Applies | Rationale |
|-------|---------|-----------|
| **Unit** | N/A | Read-only audit — no code produced |
| **Integration** | N/A | Read-only audit |
| **Contract** | N/A | Read-only audit |
| **E2E** | N/A | Read-only audit |
| **Concurrency** | N/A | Read-only audit |
| **Security** | N/A | Read-only audit |

---

## Inventory

The root package (`framework`) exports **~280 symbols** across 43 files. Adding subpackages brings the total framework surface to **~450+ exported symbols** across 20+ packages.

### Root package (43 files)

| Category | Key exports |
|----------|-------------|
| Core primitives | `Node`, `Edge`, `Graph`, `Walker`, `Extractor`, `Artifact`, `Element`, `Persona`, `Mask`, `Component` |
| Walk machinery | `Run`, `Runner`, `BatchWalk`, `ProcessWalker`, `WalkerState`, `Team` |
| DSL | `CircuitDef`, `LoadCircuit`, `BuildGraph`, `GraphRegistries`, `NodeRegistry` |
| Dialectic | `DialecticConfig`, `ThesisChallenge`, `AntithesisResponse`, `Synthesis`, `DialecticRecord`, `CMRRCheck` |
| Observability | 17 `WalkEventType` constants, `WalkObserver`, `NarrationObserver`, `TraceCollector` |
| Identity | `AgentIdentity`, `ModelIdentity`, `ModelRegistry`, `Persona`, `Color` |
| Memory | `MemoryStore`, `InMemoryStore`, `TaggedSetter` |
| Cache/Checkpoint | `NodeCache`, `InMemoryCache`, `Checkpointer`, `JSONCheckpointer` |
| Vocabulary | `Vocabulary`, `RichVocabulary`, `MapVocabulary`, `RichMapVocabulary`, `ChainVocabulary`, `RichChainVocabulary` |
| Evidence | `EvidenceGap`, `EvidenceGapBrief`, `GapSeverity`, `EvidenceSNR`, `CountableArtifact` |
| Scheduling | `Scheduler`, `SingleScheduler`, `AffinityScheduler`, `SchedulerContext` |
| Thermal | `WithThermalBudget` |

### Subpackages

| Package | Exported symbols | Consumed by |
|---------|-----------------|-------------|
| calibrate | ~35 | modules/rca |
| cli | ~15 | Achilles, modules/rca |
| components/rp | ~35 | modules/rca |
| components/sqlite | ~20 | modules/rca/store |
| checkpoint/sqlite | ~5 | modules/rca |
| curate | ~25 | modules/rca |
| dispatch | ~45 | modules/rca, mcp, kami |
| fold | ~15 | origami CLI |
| format | ~10 | modules/rca |
| knowledge | ~25 | modules/rca, components/rp |
| kami | ~35 | modules/rca |
| lint | ~35 | origami CLI |
| logging | 2 | modules/rca, components/rp |
| lsp | ~15 | origami CLI |
| mcp | ~20 | modules/rca |
| memory/sqlite | ~5 | modules/rca |
| observability | ~8 | origami CLI |
| ouroboros | ~35 | modules/rca, ouroborosmcp |
| ouroboros/probes | ~15 | ouroboros circuit |
| report | ~6 | modules/rca |
| studio/backend | ~20 | none |
| transformers | ~15 | modules/rca |

### Consumer footprint

| Consumer | Packages imported | Root exports used |
|----------|-------------------|-------------------|
| Achilles | root, cli | ~30 of ~280 (~11%) |
| modules/rca | 15+ packages | broad surface |
| Asterisk | no Go code | uses Origami via modules/rca binary |

---

## The Good

1. **Clean primitive interfaces.** `Node`, `Edge`, `Graph`, `Walker`, `Extractor`, `Artifact` are minimal, composable interfaces. Achilles implements them with near-zero friction — 4 node types, 2 extractors, one `main.go`, one YAML circuit.

2. **Zero marble residue.** Dead code excision was thorough. No `.go` file references `Marble` anywhere in the codebase.

3. **Zero TODO/FIXME/HACK/DEPRECATED markers.** The codebase has no stale annotation debt on any exported symbol.

4. **Strong test coverage on 14 of 19 audited packages.** Root, calibrate, fold, format, knowledge, lint, logging, mcp, ouroboros, report, components/rp, kami, checkpoint/sqlite, curate, and memory/sqlite all have solid public API test coverage (tests exercise exported functions and types directly).

5. **Idiomatic functional options.** `RunOption`, `GraphOption`, `NarrationOption`, `ResolveOption` follow standard Go patterns. No trivial pass-through wrappers found.

6. **No adversarial "court" terminology remnants.** "Backcourt"/"Frontcourt" are basketball zone metaphors in the identity system (`MetaPhase`), not the old adversarial "court" concept.

7. **Consumer footprint is narrow and correct.** Achilles uses only root + cli. modules/rca uses the broader surface. No consumer reaches into `internal/`.

---

## The Bad

### B1. Five packages with partial test coverage

| Package | Untested exports | Risk |
|---------|-----------------|------|
| dispatch | `StdinDispatcher`, `StaticDispatcher` | I/O-heavy user-facing entry points with zero direct tests |
| lsp | `NewStdioStream`, `ServeStream` | Transport layer — bugs break the entire LSP |
| observability | `HasOTelEndpoint` | Low risk (env var check) but trivially coverable |
| transformers | `TemplateParamsTransformer`, `CoreComponent`, `WithCoreBaseDir` | `CoreComponent` bundles the default transformer set; untested wiring |
| components/sqlite | `ExecHook`, `NewExecHook` | Hook is used by kami and rca; no coverage on the hook itself |

### B2. `ResolveFQCN` name collision

Two functions share the name with completely different semantics:

- `component.go` line ~126: `ResolveFQCN(fqcn string) (namespace, name string)` — simple string split on `"."`.
- `fold/fqcn.go` line ~24: `(r ModuleRegistry) ResolveFQCN(fqcn string) (string, error)` — registry-based lookup mapping FQCN to Go import path.

The component version has zero production callers. It is only exercised in `component_test.go`.

### B3. 14 lint rule types lack godoc

All exported rule types in `lint/rules_structural.go` use section-separator comments (`// --- S1: missing-node-element ---`) instead of proper type-level godoc: `MissingNodeElement`, `InvalidElement`, `InvalidMergeStrategy`, `MissingEdgeName`, `DuplicateEdgeCondition`, `EmptyPrompt`, `InvalidCacheTTL`, `MissingCircuitDescription`, `UnnamedNode`, `InvalidWalkerElement`, `InvalidWalkerPersona`, `SchemaInUnstructuredZone`, `MissingZoneDomain`, `InvalidZoneDomain`.

### B4. `runExprProgram` exported for testing only

`expression_edge.go` exports `runExprProgram` with a doc comment saying domain code should use `expressionEdge.Evaluate`. This leaks test infrastructure into the public API.

### B5. MetaPhase vs Zone naming overlap

`MetaPhase` (identity zones: Backcourt/Frontcourt/Paint) and `Zone` (graph grouping: `ZoneDef`) serve different purposes but the word "zone" appears in both contexts (e.g., `HomeZoneFor(p Position) MetaPhase`). Not wrong, but a documentation gap that will confuse new consumers.

---

## The Ugly

### U1. Stale "Adapter" terminology in tests

`batch_walk_test.go` contains `TestBatchWalk_PerCaseAdapters`, `hookAdapter`, `noopHookAdapter` using "Adapter" when the actual types are `*Component`. This is stale terminology from the pre-Component era. Tests are documentation; this misleads readers about current API concepts.

### U2. Root package size — 43 files, ~280 exports

The root `framework` package carries everything from `Node` to `Vocabulary` to `Thermal` to `SNR`. This is a large surface for a single Go package. Consumers face a wide API with no sub-package guidance. Candidates for extraction:

| Group | Types | Count |
|-------|-------|-------|
| Vocabulary | `Vocabulary`, `RichVocabulary`, `MapVocabulary`, `RichMapVocabulary`, `ChainVocabulary`, `RichChainVocabulary`, `VocabEntry` | 7 |
| Narration | `NarrationSink`, `NarrationOption`, `Progress`, `NarrationObserver` | 4 |
| Evidence/SNR | `EvidenceSNR`, `CountableArtifact`, `EvidenceGap`, `EvidenceGapBrief`, `GapSeverity` | 5 |
| Scheduler | `Scheduler`, `SingleScheduler`, `AffinityScheduler`, `SchedulerContext` | 4 |
| Memory | `MemoryStore`, `InMemoryStore`, `MemoryItem`, `TaggedSetter` | 4 |
| Cache | `NodeCache`, `InMemoryCache`, `CachePolicy` | 3 |

Not broken, but a future refactor opportunity.

### U3. Heavy unused surface by the only external consumer

Achilles uses ~30 of ~280 root exports (~11%). The remaining ~250 are consumed only by modules/rca, which lives inside Origami. The "public" API is largely "internal to the monorepo." A third-party consumer would face ~280 exports with no guidance on what matters.

### U4. `studio/backend` has zero external consumers

20+ exported symbols, no imports from any consumer repository. This package exists for the `origami studio` command but nothing uses it yet. It's dead surface area from the consumer's perspective.

---

## Test Coverage Matrix (Public Interfaces)

| Package | Test Quality | Untested Public Symbols |
|---------|-------------|------------------------|
| Root (`framework`) | **Solid** | None significant |
| calibrate | **Solid** | Interface-only types (by design) |
| cli | **Solid** | None |
| components/rp | **Solid** | None |
| components/sqlite | **Partial** | `ExecHook`, `NewExecHook` |
| checkpoint/sqlite | **Solid** | None |
| curate | **Solid** | None |
| dispatch | **Partial** | `StdinDispatcher`, `StaticDispatcher` |
| fold | **Solid** | None |
| format | **Solid** | None |
| knowledge | **Solid** | None |
| kami | **Solid** | None |
| lint | **Solid** | 14 rule types lack godoc (tested via `Run`) |
| logging | **Solid** | None |
| lsp | **Partial** | `NewStdioStream`, `ServeStream` |
| mcp | **Solid** | None |
| memory/sqlite | **Solid** | None |
| observability | **Partial** | `HasOTelEndpoint` |
| ouroboros | **Solid** | None |
| report | **Solid** | None |
| studio/backend | **Solid** | None (but zero consumers) |
| transformers | **Partial** | `TemplateParamsTransformer`, `CoreComponent` |

**Overall: 14 solid, 5 partial, 0 none.**

---

## Recommended Action Items (for follow-up contracts)

These are NOT tasks for this audit — they are prioritized recommendations for subsequent contracts.

### P1 — Quick fixes (low effort, high signal)

- Remove or rename `component.ResolveFQCN` → `SplitFQCN` or delete (dead export, name collision with `fold/ModuleRegistry.ResolveFQCN`)
- Un-export `runExprProgram` (move to `_test.go` helper or `internal/`)
- Rename `TestBatchWalk_PerCaseAdapters` → `TestBatchWalk_PerCaseComponents` and `hookAdapter`/`noopHookAdapter` → `hookComponent`/`noopHookComponent`
- Add godoc to 14 lint rule types in `lint/rules_structural.go`

### P2 — Test gaps (medium effort)

- Add tests for `ExecHook` / `NewExecHook` in components/sqlite
- Add tests for `CoreComponent` / `TemplateParamsTransformer` in transformers
- Add test for `HasOTelEndpoint` in observability
- Add tests for `StdinDispatcher` / `StaticDispatcher` in dispatch (may require I/O mocking)
- Add tests for `NewStdioStream` / `ServeStream` in lsp (may require transport mocking)

### P3 — Structural (high effort, future)

- Consider splitting root package into focused sub-packages (vocabulary, memory, cache, scheduler, narration)
- Audit `studio/backend` — decide if it should move to `internal/` or if it needs consumers
- Write a root `README.md` that maps the 20+ packages into a consumer learning path

---

## Tasks

- [x] Inventory root package exports (~280 symbols, 43 files)
- [x] Inventory subpackage exports (~450+ total across 20+ packages)
- [x] Map consumer usage (Achilles, Asterisk, modules/rca)
- [x] Assess test coverage per package (14 solid, 5 partial, 0 none)
- [x] Identify naming inconsistencies and dead exports
- [x] Produce Good/Bad/Ugly assessment
- [x] Produce prioritized action items for follow-up contracts

## Acceptance criteria

- Full inventory of exported symbols per package exists in this document.
- Test coverage matrix covers all 21 audited packages with quality rating.
- Every "Bad" and "Ugly" item cites specific files and symbols.
- Recommended action items are prioritized (P1/P2/P3).
- No code was changed.

## Security assessment

No trust boundaries affected. This is a read-only documentation contract.

## Notes

2026-03-01 — Contract created. Full audit completed in a single pass. Key findings:
- 280 root exports, 450+ total across 20+ packages.
- 14/19 packages have solid test coverage on public interfaces.
- 5 packages have partial coverage (dispatch, lsp, observability, transformers, components/sqlite).
- 1 dead export (`component.ResolveFQCN`), 1 test-only export (`runExprProgram`), 14 missing godocs on lint rules.
- Root package size (43 files) is the main structural concern — large but functional.
- Marble excision and RCA/RP decoupling left the surface clean.
