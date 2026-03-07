# Schematic Toolkit Extraction Log

Decision log for the `schematics/toolkit` package extraction from RCA and knowledge schematics.

## What was extracted

### From `schematics/rca/` (Phases 1-2)

| Module | Source file(s) | Rationale |
|---|---|---|
| `mapaccess` | `report_data.go`, `transformer_rca.go` | Typed `map[string]any` accessors used everywhere; zero domain specificity |
| `artifact` | `artifact.go` | Case directory layout, artifact I/O, node filename convention — generic across schematics |
| `params` | `params.go` | `LoadPriorArtifacts` bulk-loads node outputs — same pattern in any multi-node circuit |
| `component` | `component.go` | `TransformerForAllNodes`, `ExtractorForAllNodes`, `NodeNamesFromCircuit` — component wiring patterns |
| `hooks` | `hooks_inject.go` | `NewContextInjector` / `NewContextInjectorErr` — boilerplate reduction for before-hooks |
| `routing` | `routing.go` (new) | `RoutingEntry`, `RoutingLog`, `CompareRoutingLogs` — dispatch routing decision recording |
| `tuning` | `tuning.go` (new) | `QuickWin`, `TuningRunner`, `TuningReport` — tuning loop with stop conditions |
| `hitl` | `hitl.go` | `HITLResult`, `BuildHITLResult`, `LoadCheckpointState`, `RestoreWalkerState` |
| `report` | `report_data.go` | `PluralizeCount`, `SortedKeys`, `GroupByKey`, `FormatDistribution` — rendering utilities |

### From root `knowledge/` (Phases 3-5, source decoupling)

| Module | Source file(s) | Rationale |
|---|---|---|
| `source` | `knowledge/source.go` | `Source`, `SourceKind`, `ReadPolicy`, `ResolutionStatus` — generic data source descriptor |
| `source_catalog` | `knowledge/source.go` (catalog part) | `SourceCatalog` interface, `SliceCatalog` — collection abstraction |
| `source_reader` | `knowledge/reader.go` | `SourceReader` interface, `SearchResult`, `ContentEntry` — unified access |

## What stayed in schematics

### `schematics/rca/`

- `NodeArtifactFilename` wrapper — delegates to `toolkit.NodeArtifactFilename` with RCA's override map; returns `""` for unknown nodes (RCA-specific contract)
- `allNodeNames` — hardcoded fallback; `nodeNames(cd)` helper prefers `toolkit.NodeNamesFromCircuit` when a CircuitDef is available
- `rcaNodeArtifacts` map — RCA's historical node-to-filename mapping (non-convention names like `artifact.json`, `review-decision.json`)
- All domain transformers, extractors, hooks, analysis logic

### `schematics/knowledge/`

- `SourcePack`, `SourceRouter`, `DepGraph`, `VersionMatrix`, `Summarizer`, `Loader` — knowledge-schematic-specific runtime (moved from root `knowledge/` to `schematics/knowledge/`)
- `AccessRouter`, `MCPReader`, `MCPServer` — knowledge access layer

## What was deleted

- Root `knowledge/` directory — entirely eliminated. Generic types moved to toolkit; schematic-specific types moved to `schematics/knowledge/`.

## Key decisions

1. **Convention + override for artifact filenames.** Toolkit provides `<node>-result.json` convention; schematics override via map for historical names. New schematics (like Achilles) get the convention for free.

2. **Variadic CircuitDef for backward compat.** `TransformerComponent` and `HITLComponent` accept optional `*framework.CircuitDef` via variadic. Existing callers unchanged; callers with a circuit def pass it to derive node names dynamically.

3. **Interface over implementation for sources.** `SourceCatalog` and `SourceReader` are interfaces in toolkit. Implementations live in their respective schematics. This allows any schematic to define its own source access without importing knowledge internals.

4. **Zero-regression validation.** Every extraction was validated with `go test -race ./...` across Origami (50+ packages), `go build ./...` for Achilles, and `just build` for Asterisk.
