# Toolkit API Reference

Package `github.com/dpopsuev/origami/schematics/toolkit` — generic schematic runtime patterns reusable by any Origami schematic.

## Map Accessors (`mapaccess.go`)

| Function | Signature | Purpose |
|---|---|---|
| `MapStr` | `(m map[string]any, key string) string` | Extract string, zero on miss/mismatch |
| `MapFloat` | `(m map[string]any, key string) float64` | Extract float64, converts from int/int64 |
| `MapBool` | `(m map[string]any, key string) bool` | Extract bool |
| `MapInt64` | `(m map[string]any, key string) int64` | Extract int64, converts from float64/int |
| `MapStrSlice` | `(m map[string]any, key string) []string` | Extract string slice, handles `[]any` |
| `MapMap` | `(m map[string]any, key string) map[string]any` | Extract nested map |
| `MapSlice` | `(m map[string]any, key string) []any` | Extract `[]any` |
| `AsMap` | `(v any) map[string]any` | Cast `any` to map |

## Artifact I/O (`artifact.go`)

| Function | Signature | Purpose |
|---|---|---|
| `CaseDir` | `(basePath string, suiteID, caseID int64) string` | Build per-case directory path |
| `EnsureCaseDir` | `(basePath string, suiteID, caseID int64) (string, error)` | Create case dir if absent |
| `ListCaseDirs` | `(basePath string, suiteID int64) ([]string, error)` | List all case dirs under a suite |
| `NodeArtifactFilename` | `(nodeName string, overrides map[string]string) string` | Artifact filename: override map then `<node>-result.json` convention |
| `NodePromptFilename` | `(nodeName string, loopIter int) string` | Prompt filename: `prompt-<node>.md` or `prompt-<node>-loop-N.md` |
| `ReadMapArtifact` | `(dir, filename string) (map[string]any, error)` | Read JSON artifact into map |
| `WriteArtifact` | `(dir, filename string, data any) error` | Write JSON artifact |
| `WriteNodePrompt` | `(dir, nodeName string, loopIter int, content string) (string, error)` | Write prompt file |

## Parameters (`params.go`)

| Function | Signature | Purpose |
|---|---|---|
| `LoadPriorArtifacts` | `(caseDir string, nodeNames []string, artifactFn func(string) string) map[string]map[string]any` | Bulk-load prior node artifacts |

## Component Wiring (`component.go`)

| Function | Signature | Purpose |
|---|---|---|
| `TransformerForAllNodes` | `(t framework.Transformer, nodeNames []string) framework.TransformerRegistry` | Register one transformer under every node |
| `ExtractorForAllNodes` | `(factory func(string) framework.Extractor, nodeNames []string) framework.ExtractorRegistry` | Register per-node extractors via factory |
| `NodeNamesFromCircuit` | `(cd *framework.CircuitDef) []string` | Derive node names from a circuit definition |

## Hook Injection (`hooks.go`)

| Function | Signature | Purpose |
|---|---|---|
| `NewContextInjector` | `(name string, fn func(walkerCtx map[string]any)) framework.Hook` | Before-hook that injects into walker context |
| `NewContextInjectorErr` | `(name string, fn func(ctx context.Context, walkerCtx map[string]any) error) framework.Hook` | Same, but fn can return an error |

## Routing (`routing.go`)

| Type | Purpose |
|---|---|
| `RoutingEntry` | Single dispatch routing decision record |
| `RoutingLog` | Ordered sequence of routing entries with `ForCase`, `ForStep`, `Len` filters |
| `RoutingDiff` | Mismatch between expected and actual routing |

| Function | Signature | Purpose |
|---|---|---|
| `SaveRoutingLog` | `(path string, log RoutingLog) error` | Persist routing log as JSON |
| `LoadRoutingLog` | `(path string) (RoutingLog, error)` | Load routing log from JSON |
| `CompareRoutingLogs` | `(expected, actual RoutingLog) []RoutingDiff` | Diff two routing logs |

## Tuning (`tuning.go`)

| Type | Purpose |
|---|---|
| `QuickWin` | Atomic improvement step in a tuning loop |
| `TuningResult` | Before/after measurement for one QW |
| `TuningReport` | Aggregated tuning session report |
| `TuningRunner` | Executes QW sequence with stop conditions |

| Function | Signature | Purpose |
|---|---|---|
| `LoadQuickWins` | `(data []byte) []QuickWin` | Parse QW definitions from YAML |
| `NewTuningRunner` | `(qws []QuickWin, targetVal float64) *TuningRunner` | Create runner with defaults |
| `(*TuningRunner).Run` | `(baselineVal float64) TuningReport` | Execute tuning loop |

## HITL (`hitl.go`)

| Type | Purpose |
|---|---|
| `HITLResult` | Walk step outcome: interrupted or complete |

| Function | Signature | Purpose |
|---|---|---|
| `LoadCheckpointState` | `(checkpointDir, walkerID string) (*framework.WalkerState, error)` | Load checkpoint (nil if none) |
| `BuildHITLResult` | `(walker framework.Walker, walkErr error) (*HITLResult, error)` | Interpret walk outcome |
| `RestoreWalkerState` | `(walker framework.Walker, loaded *framework.WalkerState) string` | Apply checkpoint to walker, return resume node |

## Report Rendering (`report.go`)

| Function | Signature | Purpose |
|---|---|---|
| `PluralizeCount` | `(n int, singular, plural string) string` | Singular when n==1, plural otherwise |
| `SortedKeys` | `[V any](m map[string]V) []string` | Sorted keys of string-keyed map |
| `GroupByKey` | `[T any](items []T, keyFn func(T) string) map[string][]T` | Group items by extracted string key |
| `FormatDistribution` | `(counts map[string]int, labelFn func(string) string) string` | Render count map as "label (N), ..." |

## Source Types (`source.go`, `source_catalog.go`, `source_reader.go`)

| Type | Purpose |
|---|---|
| `Source` | Data source descriptor (repo, spec, doc, API) with metadata |
| `SourceKind` | Classification: `repo`, `spec`, `doc`, `api` |
| `ReadPolicy` | When to include: `always`, `conditional` |
| `ResolutionStatus` | Fetch status: `resolved`, `cached`, `degraded`, `unavailable`, `unknown` |
| `SourceCatalog` | Interface for source collections (`Sources()`, `AlwaysReadSources()`) |
| `SliceCatalog` | Slice-backed catalog implementation |
| `SourceReader` | Unified source access: `Ensure`, `Search`, `Read`, `List` |
| `SearchResult` | Single search hit |
| `ContentEntry` | File/directory in a source listing |

## Consumers

- **RCA schematic** (`schematics/rca/`) — uses all modules; RCA-specific overrides for artifact filenames and node names
- **Knowledge schematic** (`schematics/knowledge/`) — uses `Source`, `SourceCatalog`, `SourceReader`
- **Achilles** (`github.com/dpopsuev/achilles`) — uses `NodeNamesFromCircuit`, `PluralizeCount`, `GroupByKey`, `NodeArtifactFilename`
