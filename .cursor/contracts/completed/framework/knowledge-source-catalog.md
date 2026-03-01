# Contract — KnowledgeSourceCatalog + KnowledgeSourceRouter

**Status:** complete  
**Goal:** Rename `Workspace` to `KnowledgeSourceCatalog`, `Repo` to `Source` (with Kind/Tags fields), and introduce `KnowledgeSourceRouter` as a batteries-included routing struct for directing circuit steps to relevant knowledge sources.  
**Serves:** Framework Maturity

## Contract rules

- Zero domain imports — all types and routing logic are framework-generic.
- Consumers (Asterisk, Achilles) configure sources and routing rules; framework provides the engine.
- Backward compatibility: provide a deprecation alias or migration guide for `Workspace`/`Repo`.

## Context

- **Origin:** Asterisk Phase 5a R11 analysis revealed that `selectRepoByHypothesis` (hardcoded Purpose keyword matching) breaks across scenarios. The root cause is that repo routing is domain-specific but implemented with brittle keyword rules instead of structured metadata.
- **Design decision:** `Workspace` is a Cursor IDE term that leaked into the framework. The concept is really "a catalog of knowledge sources the circuit can consult." Sources include repos, specs, docs, APIs — not just code repositories.
- **Asterisk dependency:** `phase-5a-v2-analysis.md` (Asterisk) depends on this contract for the long-term M9/M10 fix. Tactical fix uses `RepoConfig.RelevantToRCAs` in the interim.
- **Consumer impact:** Asterisk has 11 files importing `workspace.*`. Achilles has 0.

### Current architecture

```
workspace/
  workspace.go    Workspace{Repos []Repo}
                  Repo{Path, URL, Name, Purpose, Branch}
  loader.go       LoadFromPath() -> *Workspace
  loader_test.go
```

Consumers call `workspace.LoadFromPath()`, get a flat `Workspace`, pass it through the circuit. No routing intelligence — consumers implement their own (Asterisk: `selectRepoByHypothesis`).

### Desired architecture

```
knowledge/
  source.go       Source{Name, Kind, URI, Purpose, Tags}
                  SourceKind: repo, spec, doc, api
                  KnowledgeSourceCatalog{Sources []Source}
  router.go       KnowledgeSourceRouter{catalog, rules}
                  RouteRule interface
                  RouteRequest{Component, Hypothesis, Step, Tags}
                  TagMatchRule (batteries-included)
  loader.go       LoadFromPath() -> *KnowledgeSourceCatalog
  loader_test.go
  router_test.go
```

Consumers configure a `KnowledgeSourceRouter` with domain-specific `RouteRule` implementations. Framework provides `TagMatchRule` as a batteries-included rule that selects sources by tag matching.

## FSC artifacts

| Artifact | Target | Compartment |
|----------|--------|-------------|
| KnowledgeSourceCatalog design reference | `docs/` | domain |
| Glossary: KnowledgeSourceCatalog, KnowledgeSourceRouter, Source, SourceKind | `glossary/` | domain |

## Execution strategy

1. **Define types** — `Source`, `SourceKind`, `KnowledgeSourceCatalog` in `knowledge/source.go`
2. **Define router** — `KnowledgeSourceRouter`, `RouteRule`, `RouteRequest`, `TagMatchRule` in `knowledge/router.go`
3. **Migrate loader** — update `LoadFromPath` to return `*KnowledgeSourceCatalog`; support both old (`repos:`) and new (`sources:`) YAML keys
4. **Update tests** — migrate `loader_test.go`, add `router_test.go`
5. **Update mask.go** — optional string references
6. **Consumer migration guide** — document the rename for Asterisk (11 files)

## Coverage matrix

| Layer | Applies | Rationale |
|-------|---------|-----------|
| **Unit** | yes | Source types, router logic, tag matching, loader |
| **Integration** | no | No cross-boundary changes |
| **Contract** | yes | Public API change: Workspace -> KnowledgeSourceCatalog |
| **E2E** | no | Framework-only; consumer E2E validates after migration |
| **Concurrency** | no | No shared state |
| **Security** | no | No trust boundaries affected |

## Tasks

- [x] **Define Source types** — `Source`, `SourceKind`, `KnowledgeSourceCatalog` in `knowledge/source.go`
- [x] **Define router types** — `KnowledgeSourceRouter`, `RouteRule` interface, `RouteRequest`, `TagMatchRule`, `RequestTagMatchRule` in `knowledge/router.go`
- [x] **Implement TagMatchRule** — select sources where `Tags[key]` matches request fields
- [x] **Migrate loader** — `LoadFromPath` returns `*KnowledgeSourceCatalog`; backward-compatible YAML/JSON parsing of `repos:` and `sources:`
- [x] **Update tests** — 8 loader tests (new/legacy, YAML/JSON, detect, empty) + 10 router tests (tag match, multi-required, no rules, no match, nil/empty catalog, multi-rule, request tag match, defensive copy)
- [x] **Update mask.go** — `workspace_repos_available` → `knowledge_sources_available`
- [x] **Consumer migration guide** — documented in Notes section below
- [x] Validate (green) — all tests pass, acceptance criteria met.
- [x] Tune (blue) — refactor for quality. No behavior changes.
- [x] Validate (green) — all tests still pass after tuning.

## Acceptance criteria

- **Given** the `knowledge/` package defines `Source`, `KnowledgeSourceCatalog`, and `KnowledgeSourceRouter`,
- **When** a consumer creates a router with tag-based rules and calls `Route(request)`,
- **Then** the router returns only sources whose tags match the request fields.

- **Given** a YAML file with `repos:` (old format),
- **When** `LoadFromPath` is called,
- **Then** it returns a valid `KnowledgeSourceCatalog` with `Kind: "repo"` for each source.

- **Given** a YAML file with `sources:` (new format) including mixed kinds (repo, spec, doc),
- **When** `LoadFromPath` is called,
- **Then** it returns a `KnowledgeSourceCatalog` with correct `Kind` for each source.

- **Given** the router has no matching rules for a request,
- **When** `Route` is called,
- **Then** it returns all non-excluded sources (safe default).

## Security assessment

No trust boundaries affected.

## Notes

2026-02-24 23:00 — All implementation tasks complete. `knowledge/` package created with source types, router, loader, and 18 tests. `workspace/` deprecated with doc comments. `mask.go` meta key updated. Validate/tune remaining.

### Consumer migration guide

| Old (workspace) | New (knowledge) |
|-----------------|-----------------|
| `workspace.Workspace` | `knowledge.KnowledgeSourceCatalog` |
| `workspace.Repo` | `knowledge.Source` |
| `workspace.LoadFromPath` | `knowledge.LoadFromPath` |
| `workspace.Load` | `knowledge.Load` |
| `Repo.Path` / `Repo.URL` | `Source.URI` |
| `Repo.Purpose` | `Source.Purpose` + `Source.Tags` |
| `ws.Repos` | `catalog.Sources` |

**Asterisk files to migrate (11):**

| File | Usage |
|------|-------|
| `cmd/asterisk/cmd_analyze.go` | `LoadFromPath`, `ws.Repos` iteration |
| `cmd/asterisk/cmd_cursor.go` | `LoadFromPath`, passes `ws` to `RunStep` |
| `internal/calibrate/adapter.go` | `SetWorkspace(ws *workspace.Workspace)` interface |
| `internal/calibrate/adapt/cursor.go` | `ws *workspace.Workspace` field + setter |
| `internal/calibrate/adapt/routing.go` | `SetWorkspace` in `RoutingRecorder` |
| `internal/calibrate/adapt/routing_test.go` | `SetWorkspace` in `fakeStoreAwareAdapter` |
| `internal/calibrate/workspace_convert.go` | `ScenarioToWorkspace` constructs `Workspace` + `Repo` |
| `internal/investigate/analyze.go` | `AnalyzeWithWorkspace` parameter |
| `internal/orchestrate/params.go` | `BuildParams` parameter, `ws.Repos` usage |
| `internal/orchestrate/params_test.go` | `workspace.Workspace{Repos: ...}` literals |
| `internal/orchestrate/runner.go` | `RunStep` parameter |

**Migration approach:** Replace `workspace.*` imports with `knowledge.*`, update struct field references (`Repos` → `Sources`, `Path`/`URL` → `URI`). `workspace_convert.go` becomes `catalog_convert.go` returning `*KnowledgeSourceCatalog`.

2026-02-24 21:30 — Contract drafted. Motivated by Asterisk Phase 5a R11 analysis: `selectRepoByHypothesis` uses Purpose keyword matching that breaks across scenarios. The framework should provide structured routing via tags/metadata instead. `Workspace` renamed to `KnowledgeSourceCatalog` to remove Cursor IDE terminology from the framework. `KnowledgeSourceRouter` wraps the catalog with configurable routing rules. Asterisk has 11 files to migrate; Achilles has 0.
2026-02-24 — Contract complete. All implementation tasks done: `KnowledgeSourceCatalog` and `Source` types implemented in `knowledge/` package, `KnowledgeSourceRouter` with tag-based routing, YAML/JSON serialization, comprehensive test coverage. Asterisk consumer migration deferred to a separate contract.
