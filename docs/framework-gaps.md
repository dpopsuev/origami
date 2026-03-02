# Framework Gaps Inventory

Missing Origami primitives needed to support zero-Go migration of consumer code. Gaps are ordered by dependency — resolving top gaps unblocks the most downstream work.

## Summary

| # | Gap | Blocks | Severity | Est. effort |
|---|-----|---------------|----------|-------------|
| G1 | Declarative extractor DSL | `llm-extract` | Critical | Medium |
| G2 | Declarative transformer DSL | `context-builder` | Critical | Medium |
| G3 | YAML-configured provider chains | `dispatch` | High | Medium |
| G4 | Hook persistence DSL | `persist` | High | Small |
| G5 | Scorer registry + evaluation engine | `score` | High | Medium |
| G6 | Report template engine | `report` | Medium | Medium |
| G7 | `NodeDef meta:` field | All components | High | Small |
| G8 | `origami fold` | CLI, MCP config | Medium | Large |
| G9 | Generic transformers | `ingest` migration | Medium | Medium |
| G10 | Step schema DSL | MCP config | Low | Small |
| G11 | Calibration-as-pipeline | `score`, `report` | Medium | Large |

---

## G1: Declarative extractor DSL

**Blocks:** `llm-extract` component

**Current state:** Extractors are Go types implementing `framework.Extractor` interface (`Name()`, `Extract(ctx, input) (any, error)`). Consumers write Go code per extractor. Asterisk has `StepExtractor[T]` — a generic JSON unmarshaler. Achilles has `GovulncheckExtractor` and `ClassifyExtractor`.

**Gap:** No way to define an extractor in YAML. The extractor's logic (parse JSON, validate schema, map fields) is always Go.

**Proposed solution:**

```yaml
extractors:
  - name: recall-extractor
    type: json-schema          # built-in: json-schema, regex, govulncheck
    schema:
      type: object
      properties:
        test_name: {type: string}
        status: {type: string}
      required: [test_name, status]
    on_error: partial          # partial | fail | skip
```

Built-in extractor types handle common patterns (JSON schema validation, regex capture, structured text). Domain-specific extractors remain Go but are registered via `adapter.yaml`.

---

## G2: Declarative transformer DSL

**Blocks:** `context-builder` component

**Current state:** Transformers are Go types implementing `framework.Transformer` (`Name()`, `Transform(ctx, tc) (any, error)`). Asterisk has `ContextBuilder` (reads store/envelope/catalog) and `PromptFiller` (fills Go text/template).

**Gap:** No way to define a transformer chain in YAML. The 500-line `BuildParams()` function assembles template parameters from multiple data sources — this logic is currently pure Go.

**Proposed solution:**

```yaml
transformers:
  - name: context-builder
    type: template-params      # built-in: template-params, jq, map
    sources:
      - walker_state           # prior artifacts, visited nodes
      - store: {table: cases, where: "id = {{ case_id }}"}
      - file: {path: "{{ prompt_dir }}/{{ step }}.md"}
    output: params

  - name: prompt-filler
    type: go-template
    template: "{{ prompt_dir }}/{{ step }}.md"
    params: "{{ params }}"
    output: prompt
```

The `template-params` type defines which data sources feed into the template. The `go-template` type renders the filled template. Both are YAML-configured, not Go-coded.

---

## G3: YAML-configured provider chains

**Blocks:** `dispatch` component

**Current state:** `dispatch.MuxDispatcher` routes to providers (Cursor, OpenAI) with signal bus and cost tracking. But provider selection, fallback order, and adapter configuration are wired in Go (`LLMAdapter`, `BasicAdapter`, `StubAdapter`).

**Gap:** No way to define the provider chain in YAML. The `adapt/basic.go` (577 LOC) is a heuristic fallback adapter with PTP-specific keyword lists — domain logic that should be YAML data, not Go code.

**Proposed solution:**

```yaml
dispatch:
  providers:
    - name: cursor
      type: llm
      config: {model: "claude-sonnet-4-20250514"}
    - name: heuristic
      type: rules
      rules_file: heuristics.yaml    # keyword lists, component maps
    - name: stub
      type: static
      artifacts_dir: stubs/
  fallback: next                      # try providers in order
  routing_log: true                   # enable RoutingRecorder
```

The `rules` provider type loads heuristic rules from YAML (keyword lists, pattern matches, component mappings). This replaces the 577 LOC of `basic.go` with ~100 lines of YAML.

---

## G4: Hook persistence DSL

**Blocks:** `persist` component

**Current state:** `framework.HookRegistry` with `NewHookFunc()` allows registering Go functions on step completion. Asterisk has 5 hooks in `hooks.go` (32 LOC) that delegate to `apply*Effects()` functions in `runner.go`.

**Gap:** Hooks are Go functions. The persistence logic (which fields to save, which table, which conditions) is coded in Go, not declared in YAML.

**Proposed solution:**

```yaml
hooks:
  - name: persist-recall
    on: step_complete
    when: "{{ step }} == 'recall'"
    actions:
      - file_write:
          path: "{{ case_dir }}/recall.json"
          data: "{{ artifact }}"
      - sqlite_exec:
          adapter: asterisk-store
          query: "UPDATE cases SET symptom_id = ?, updated_at = ? WHERE id = ?"
          params: ["{{ artifact.symptom_id }}", "{{ now }}", "{{ case_id }}"]
```

Hook actions are YAML-declared operations: file writes, SQLite queries, HTTP calls. The `sqlite_exec` action uses the sqlite adapter from the previous contract. Condition expressions use the same `when:` syntax as pipeline edges.

---

## G5: Scorer registry + evaluation engine

**Blocks:** `score` component

**Current state:** Asterisk's `metrics.go` (646 LOC) has 21 hand-coded scorer functions. The `ScoreCard` YAML defines thresholds and weights but the scorers themselves are Go functions.

**Gap:** No framework-level scorer registry. Each consumer re-implements scoring from scratch. The ScoreCard evaluation engine exists in `calibrate/` but requires Go scorer registration.

**Proposed solution:**

Enhance `calibrate/` with:
1. **Scorer registry** — register named scorer functions at startup
2. **Built-in scorers** — common patterns: accuracy (`correct/total`), correlation (`pearson(predicted, actual)`), rate (`count/total` with threshold)
3. **ScoreCard evaluator** — load ScoreCard YAML, invoke registered scorers, compute weighted aggregate

```yaml
scorecard:
  metrics:
    - id: M1
      name: Defect Type Accuracy
      scorer: accuracy
      params: {predicted: defect_type, expected: expected_defect_type}
      threshold: {min: 0.85}
      weight: 0.20
```

Domain-specific scorers (like M5 serial killer detection) remain Go but are registered by name. Common patterns become built-in.

---

## G6: Report template engine

**Blocks:** `report` component

**Current state:** Asterisk has 5 report types across 5 files (~865 LOC), each hand-coded with go-pretty tables and string formatting. Achilles has a terminal report in `reportNode`.

**Gap:** No framework-level report templating. Each report format is a Go function.

**Proposed solution:**

```yaml
reports:
  - name: calibration-report
    format: terminal
    sections:
      - type: table
        title: "Outcome Metrics"
        columns: [ID, Metric, Value, Detail, Pass, Threshold]
        data: "{{ metrics | where: category == 'outcome' }}"
      - type: table
        title: "Per-case Breakdown"
        columns: [Case, Test, Defect, DT, Path]
        data: "{{ cases }}"
```

Report sections are composable YAML blocks: tables, text, Markdown, headers. The engine uses `format/` (go-pretty) for terminal rendering and Go text/template for Markdown. Domain tools define report structure in YAML; the engine handles rendering.

---

## G7: `NodeDef meta:` field

**Blocks:** All components (configurable nodes)

**Current state:** `NodeDef` in YAML defines `name`, `element`, `family`, `zone`. Nodes receive the `NodeDef` in their `Process()` call but cannot read arbitrary configuration from the YAML.

**Gap:** Nodes cannot be parameterized from YAML. For example, a `persist` node needs to know the file path pattern and SQLite table; currently this is hard-coded in Go.

**Proposed solution:**

```yaml
nodes:
  - name: recall
    element: earth
    family: recall
    meta:
      prompt_template: "prompts/recall.md"
      persist_to: "cases"
      extractor: recall-extractor
```

Add `Meta map[string]any` to `NodeDef`. Nodes access configuration via `nc.NodeDef().Meta["key"]`. This is the smallest gap with the highest leverage — it makes all other components configurable from YAML.

---

## G8: `origami fold`

**Blocks:** CLI (`cmd/asterisk/`, 2,032 LOC), MCP config (`internal/mcpconfig/`, 1,249 LOC)

**Current state:** Asterisk has a Cobra CLI with 10 commands and an MCP server, all wired in Go. See `origami-fold-concept.md` in notes.

**Gap:** No way to compile a pure-YAML project into a standalone binary. The CLI entry point, command registration, adapter imports, and MCP wiring require Go code.

**Proposed solution:** `origami fold` reads an `origami.yaml` manifest that declares:
- Pipeline files to embed
- Adapters to link (RP, SQLite, etc.)
- CLI commands (mapped to pipeline runs)
- MCP server configuration (step schemas, tools)

Generates a `main.go` + `go.mod`, builds, produces a binary. See `notes/origami-fold-concept.md` for full design.

**Effort:** Large — involves code generation, adapter discovery, and build toolchain. Deferred to post-PoC implementation.

---

## G9: Generic transformers

**Blocks:** Ingest migration (`adapters/ingest/`, 634 LOC)

**Current state:** Asterisk's ingest pipeline has 6 nodes: fetch_launches, parse_failures, match_symptoms, deduplicate, create_candidates, notify_review. Each is a Go `framework.Node` with domain logic.

**Gap:** Common data operations (pattern matching, deduplication, JSON file I/O) are implemented as custom Go nodes instead of reusable YAML-configured transformers.

**Proposed solution:** Add to `transformers/`:
- `match.pattern` — regex/glob pattern matching on structured data
- `dedup.by_key` — deduplicate records by a key field
- `file.write_json` / `file.read_json` — JSON file I/O
- `http.fetch` — HTTP GET/POST with response parsing

```yaml
nodes:
  - name: match_symptoms
    element: fire
    meta:
      transformer: match.pattern
      config:
        field: error_message
        patterns_from: symptoms.yaml
        output: matched_symptom_id
```

---

## G10: Step schema DSL

**Blocks:** MCP config migration (`internal/mcpconfig/`, partial)

**Current state:** Asterisk's MCP server defines F0-F6 step schemas in Go (`asteriskStepSchemas` map). Each schema lists field names, types, and required flags.

**Gap:** Step schemas are Go data structures, not YAML.

**Proposed solution:**

```yaml
step_schemas:
  F0_RECALL:
    fields:
      - name: test_name
        type: string
        required: true
      - name: symptom_id
        type: string
```

The MCP server loads step schemas from YAML. Smallest gap, lowest priority — already mostly declarative, just needs YAML serialization.

---

## G11: Calibration-as-pipeline

**Blocks:** `score` and `report` components (full DSL)

**Current state:** Asterisk's calibration runner (`cal_runner.go`, 684 LOC) and parallel executor (`parallel.go`, 752 LOC) orchestrate multi-case calibration. This is procedural Go code that drives the pipeline, scores results, and formats reports.

**Gap:** Calibration is a Go procedure, not a declarative pipeline. The calibration "outer loop" (load scenario, run cases, score, aggregate, report) could itself be expressed as an Origami pipeline.

**Proposed solution:** Define a meta-pipeline:

```yaml
pipeline: calibrate
nodes:
  - name: load_scenario
  - name: run_cases        # fan-out: one case = one walk of the inner pipeline
  - name: score_results    # uses ScoreCard
  - name: aggregate        # M19, M20
  - name: report           # uses report template
```

The calibration pipeline walks the scoring/reporting components. Cases are fan-out sub-walks of the domain pipeline (RCA or security scan).

**Effort:** Large — requires meta-pipeline support (pipeline-walks-pipeline). Deferred to post-PoC implementation.

---

## Resolution path

| Phase | Gaps addressed | Outcome |
|-------|---------------|---------|
| Phase 2a (foundation) | G7 (`NodeDef meta:`), G4 (hook DSL) | Configurable nodes, declarative persistence |
| Phase 2b (extraction) | G1 (extractor DSL), G2 (transformer DSL) | `llm-extract` + `context-builder` components |
| Phase 2c (dispatch) | G3 (provider chains) | `dispatch` component |
| Phase 2d (scoring) | G5 (scorer registry) | `score` component |
| Phase 2e (reporting) | G6 (report templates) | `report` component |
| Phase 3 (infrastructure) | G9 (generic transformers), G10 (step schema DSL) | Ingest + MCP migration |
| Phase 4 (compilation) | G8 (`origami fold`) | CLI + MCP wiring migration |
| Phase 5 (meta) | G11 (calibration-as-pipeline) | Full calibration DSL |
