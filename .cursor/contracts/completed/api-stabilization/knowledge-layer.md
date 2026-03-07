# Contract — knowledge-layer

**Status:** complete  
**Goal:** The knowledge schematic gains a synthesis layer — producing derived domain context artifacts from multiple resolved sources — and legacy "workspace" naming is eliminated.  
**Serves:** Containerized Runtime

## Contract rules

- Deterministic first. Structural synthesis (metadata merge, Markdown formatting) is deterministic. LLM-enriched synthesis is deferred to a follow-up contract.
- Derived artifacts are transient. Produced per-investigation, injected into prompts, never persisted to git.
- Existing source access (Ensure, Search, Read, List) is unchanged. Synthesis is additive.

## Context

**Merged from two contracts:**

- `knowledge-sources-github` (Asterisk draft) — 95% absorbed by prior contracts (`domain-separation-container`, `decoupled-schematics`, `knowledge-source-evolution`). Source packs, GitHub connector, code injection hooks, RepoCache, auth — all implemented. Only the workspace→sources rename remained.
- `knowledge-synthesis` (Origami draft) — the synthesis gap: `schematics/knowledge/` is a source access layer with zero synthesis capability. It can fetch water from 5 wells but can't blend them.

### What exists today

| Capability | Status | Where |
|---|---|---|
| Load source packs | done | `source_pack.go` |
| Route sources by tags | done | `source_router.go` |
| Resolve branch patterns | done | `branch_resolver.go` |
| Access content (Ensure, Search, Read, List) | done | `access_router.go`, `mcp_reader.go` |
| GitHub connector (clone, search, read) | done | `connectors/github/` |
| Docs connector (HTTP fetch) | done | `connectors/docs/` |
| Code injection hooks (tree, search, read) | done | `schematics/rca/hooks_inject.go` |
| MCP tools (ensure, search, read, list) | done | `mcp_server.go` |
| Version → branch/tag mapping | done | `version_matrix.go` |
| Dependency ordering | done | `depgraph.go` |
| Truncate/summarize single content | done | `summarizer.go` |
| **Synthesize derived artifacts from multiple sources** | **missing** | — |
| **MCP tool for synthesis** | **missing** | — |
| **Legacy "workspace" naming cleanup** | **remaining** | `hooks_inject.go` |

## FSC artifacts

| Artifact | Target | Compartment |
|----------|--------|-------------|
| Extraction log | `notes/knowledge-layer-log.md` | domain |

## Execution strategy

Four phases, strictly ordered. Each phase leaves the build green.

**Phase 1 — Contract housekeeping.** Merge the two source contracts, update indexes and manifests.

**Phase 2 — Workspace→sources rename.** Merge `inject.workspace` into `inject.sources`, rename `KeyParamsWorkspace`, update `ParamsFromContext`.

**Phase 3 — Synthesis types + StructuralSynthesizer.** Define `DerivedArtifact`, `Synthesizer` interface, `StructuralSynthesizer`. Unit tests with mock `SourceReader`.

**Phase 4 — MCP tool.** Register `knowledge_synthesize` in `mcp_server.go`. Wire `Synthesizer` + `PackResolver` + `SourceReader`.

## Coverage matrix

| Layer | Applies | Rationale |
|-------|---------|-----------|
| **Unit** | yes | Synthesizer implementations, DerivedArtifact construction, budget allocation |
| **Integration** | no | No external dependencies in synthesis path; MCP tool tested via unit test |
| **Contract** | yes | `Synthesizer` interface contract |
| **E2E** | no | Deferred to LLM-enriched follow-up |
| **Concurrency** | no | Synthesis is per-investigation, single-threaded |
| **Security** | yes | Source content sanitized before synthesis output |

## Tasks

### Phase 1 — Contract housekeeping

- [x] P1.1 — Create merged contract, mark `knowledge-sources-github` as absorbed, delete `knowledge-synthesis`, update indexes/manifests.

### Phase 2 — Workspace→sources rename

- [x] P2.1 — Merge `inject.workspace` into `inject.sources` in `hooks_inject.go`. Fold `buildSourceParams` logic into the sources hook.
- [x] P2.2 — Rename `KeyParamsWorkspace` → consolidate with `KeyParamsSources`. Update `ParamsFromContext`.
- [x] P2.3 — Update tests referencing `inject.workspace`.
- [x] P2.4 — Validate: `go test -race ./schematics/rca/...` green.

### Phase 3 — Synthesis types + StructuralSynthesizer

- [x] P3.1 — Define `DerivedArtifact`, `SynthesisOpts`, `Synthesizer` interface in `synthesizer.go`.
- [x] P3.2 — Implement `StructuralSynthesizer` in `structural_synthesizer.go`: resolve sources, extract metadata, format Markdown sections, budget-fit via `BudgetAllocator` + `Summarizer`.
- [x] P3.3 — Unit tests in `synthesizer_test.go`: mock SourceReader, verify sections, budget enforcement, empty sources, nil reader.
- [x] P3.4 — Validate: `go test -race ./schematics/knowledge/...` green.

### Phase 4 — MCP tool

- [x] P4.1 — Register `knowledge_synthesize` MCP tool in `mcp_server.go`.
- [x] P4.2 — Validate: `go test -race ./...` green across Origami, `go build` green in Asterisk + Achilles.
- [x] P4.3 — Tune (blue) — refactor for quality, no behavior changes.
- [x] P4.4 — Validate (green) — all tests still pass after tuning.

## Acceptance criteria

```gherkin
Given a source pack with 4 repos and 2 doc sources
When StructuralSynthesizer.Synthesize() is called
Then a DerivedArtifact is produced with:
  - Kind = "domain-context"
  - Sections containing "component-map", "source-index"
  - Content fitting within the specified token budget
  And no LLM call is made (deterministic)

Given the knowledge_synthesize MCP tool
When called with a source pack path and token budget
Then it returns a DerivedArtifact JSON response
  And the response includes all structural sections

Given the RCA hooks registry
When InjectHooksWithOpts is called
Then no hook named "inject.workspace" is registered
  And "inject.sources" provides both repo metadata and always-read content
```

## Security assessment

| OWASP | Finding | Mitigation |
|-------|---------|------------|
| A03:2021 Injection | Source content injected into synthesis output | Sanitize via existing Summarizer truncation; no raw HTML |

## Notes

2026-03-07 — P4.3-P4.4 (tune + validate). Extracted `charsPerToken` constant from magic number in `fitBudget`. Hoisted `sectionTitles` map to package level. Added nil-guard on `SynthesizeToolOpts.Synthesizer` defaulting to `StructuralSynthesizer`. All tests green, Origami/Achilles build clean. Contract complete.

2026-03-07 — P2-P4 executed. `inject.workspace` eliminated: merged into `inject.sources`, `KeyParamsWorkspace` removed, `ParamsFromContext` reads unified `KeyParamsSources`. Hook count reduced from 8 to 7 (NilStore) / 13 to 12 (WithStore). Synthesis layer added: `DerivedArtifact`, `Synthesizer` interface, `StructuralSynthesizer` with 3 section builders (component-map, source-index, version-info), budget fitting via `Summarizer`. 11 unit tests all green. `knowledge_synthesize` MCP tool registered in `mcp_server.go` and wired into `cmd/serve/main.go`. All tests pass: `schematics/rca`, `schematics/knowledge`, `schematics/toolkit`. Achilles builds clean. Zero `inject.workspace`/`KeyParamsWorkspace` references remaining.

2026-03-05 — Contract created by merging `knowledge-sources-github` (Asterisk, 95% absorbed by prior work) and `knowledge-synthesis` (Origami). Discovery: GitHub connector, source packs, code injection hooks, MCP tools, RepoCache — all already implemented. Remaining work: workspace→sources rename + synthesis layer. LLM-enriched synthesis deferred.
