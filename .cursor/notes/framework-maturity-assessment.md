# Framework Maturity Assessment

**Date:** 2026-02-18
**Version:** v1.0.0 (post DSL C1–C5)

## Proven capabilities (Green border)

Validated by two independent consumers (Asterisk, Achilles):

| Capability | Proven by |
|---|---|
| YAML Pipeline DSL (`PipelineDef`, `NodeDef`, `EdgeDef`) | Both |
| Directed graph walk (`Walk`, `WalkTeam`) | Both |
| Expression edges (`when:` via expr-lang/expr) | Both |
| Transformer interface + 4 built-in transformers (llm, http, jq, file) | Unit tests |
| Hook interface + `after:` hook walk integration | Asterisk (5 store hooks) |
| ArtifactSchema validation (`validatingWalker`) | Both |
| WalkObserver + LogObserver + TraceCollector | Both |
| Variable system (`vars:`, `MergeVars`) | Asterisk |
| `Run()` / `Validate()` Go API | Achilles |
| `origami run` / `origami validate` CLI | Unit tests |
| Mermaid rendering (`Render`) | Both |
| Dispatch package (Stdin, File, Mux, Batch) | Asterisk |
| Pull-based concurrent execution (MuxDispatcher + 4 Cursor subagents) | Asterisk |

## Known gaps

### 1. Input resolution not wired into Walk loop

`ResolveInput` exists in `vars.go` and is tested in isolation, but the Walk loop in `graph.go` does not call it. `transformerNode.Process()` uses `nc.PriorArtifact.Raw()` instead of resolving `${node.output}` references from `WalkerState.Outputs`.

**Contract:** `origami-green-border-integrity` (gate)

### 2. Prompt rendering not wired into transformerNode

`RenderPrompt` exists in `vars.go` and is tested in isolation, but `transformerNode.Process()` passes the raw `NodeDef.Prompt` string without template rendering. `TemplateContext` (vars, inputs, outputs) is not assembled.

**Contract:** `origami-green-border-integrity` (gate)

### 3. PoC batteries incomplete

Not every BYO interface has a trivial implementation:
- No `HTTPDispatcher` for OpenAI-compatible APIs
- No `StaticTokenAuth` for bearer token injection
- No `JSONCheckpointer` for WalkerState serialization
- No `MemoryStore` for `curate.Store`

**Contract:** `origami-poc-batteries` (quick-win)

### 4. CLI integration test missing

`origami run` and `origami validate` have unit tests but no end-to-end integration test that exercises a complete YAML pipeline through the CLI binary.

**Contract:** `origami-green-border-integrity` (gate)

### 5. WalkTeam not accessible via Run API

`graph.WalkTeam` exists but `Run()` has no `WithTeam()` option. Consumers wanting multi-walker execution must build the graph manually.

**Contract:** `origami-walkteam-run-api` (quick-win)

### 6. Adversarial Dialectic terminology not fully applied

Code still uses legacy names (`CourtConfig`, `ShouldActivate`, `MaxHandoffs`, `MaxRemands`). Target: `DialecticConfig`, `NeedsAntithesis`, `MaxTurns`, `MaxNegations`.

**Contract:** `origami-adversarial-dialectic` (gate)

## Trajectory

See `strategy/origami-vision.mdc` for the full vision including BYO architecture, execution model evolution, and scope borders.
