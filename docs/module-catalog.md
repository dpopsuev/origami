# Marble Catalog

First user-discovered marbles, extracted from Asterisk's `adapters/rca/` (14,600 LOC) and cross-validated against Achilles's pipeline. Marbles are composable YAML subgraphs that any Origami analysis tool can import.

## Design principles

1. **Ansible collection model.** Marbles are to Origami what Ansible Roles are to Ansible — reusable building blocks, not one-off glue.
2. **Two-consumer validation.** Every marble must work for both Asterisk (passive CI post-mortem) and Achilles (proactive security probing). If a marble only fits one tool, the abstraction is wrong.
3. **DSL-first.** The marble's public interface is YAML. Consumer code configures marbles via YAML, never by writing Go.
4. **Zero domain imports.** Marbles live in Origami and must not import from Asterisk, Achilles, or any consumer.

## Marble summary

| # | Marble | Purpose | Asterisk source | Achilles counterpart |
|---|--------|---------|-----------------|----------------------|
| 1 | `llm-extract` | Prompt template + output schema -> parsed structured result | `extractor.go`, `adapt/llm.go` | `extractors.go` (govulncheck, classify) |
| 2 | `context-builder` | Walker state + templates -> filled prompt variables | `transformers.go`, `params.go`, `template.go`, `catalog_convert.go` | Implicit in node logic |
| 3 | `persist` | Step artifact -> file/SQLite persistence chain | `artifact.go`, `hooks.go` | None (future: finding storage) |
| 4 | `score` | Results + scorecard -> metric values | `metrics.go`, `evidence_gap.go`, `cluster.go` | `assessNode` risk scoring |
| 5 | `report` | Scored results -> formatted output | `report.go`, `rca_report.go`, `transcript.go`, `briefing.go`, `tokimeter.go` | `reportNode` terminal report |
| 6 | `dispatch` | Intent -> provider -> fallback chain with routing log | `adapt/basic.go`, `adapt/llm.go`, `adapt/routing.go`, `adapt/stub.go` | Direct subprocess call |

---

## 1. `llm-extract`

**Purpose:** Send a prompt to an LLM (or deterministic tool) and parse the structured response into a typed artifact.

**Interface:**

```yaml
marble: llm-extract
inputs:
  prompt: string        # filled prompt text
  schema: object        # expected output JSON schema
  provider: string      # LLM provider name (or "stub", "govulncheck")
outputs:
  artifact: object      # parsed structured result matching schema
  raw: string           # raw provider response (for debugging)
  tokens: object        # {input: int, output: int} token counts
```

**Asterisk usage:**
- `StepExtractor[T]` (extractor.go) — generic JSON parser wrapping `framework.Extractor`
- `LLMAdapter.SendPrompt()` (adapt/llm.go) — dispatches to Cursor/OpenAI via `dispatch.MuxDispatcher`
- `BasicAdapter` (adapt/basic.go) — heuristic fallback when LLM unavailable
- `StubAdapter` (adapt/stub.go) — returns pre-authored artifacts for calibration

**Achilles usage:**
- `GovulncheckExtractor` — parses govulncheck JSON stream (non-LLM)
- `ClassifyExtractor` — deterministic deduplication and severity assignment

**Generalization:** The marble must support both LLM and deterministic extraction. The provider is pluggable: `cursor`, `openai`, `govulncheck`, `stub`, or any registered provider. The schema defines the expected output shape; extraction fails if the response doesn't match.

**Origami location:** `marbles/extract/` or integrated into framework extractor DSL.

---

## 2. `context-builder`

**Purpose:** Assemble prompt variables from walker state, prior artifacts, domain data, and templates.

**Interface:**

```yaml
marble: context-builder
inputs:
  state: walker_state    # current walker state (visited nodes, artifacts)
  templates:
    - path: string       # template file path (Go text/template)
  sources:               # domain-specific data sources
    - type: string       # "store", "envelope", "catalog", "workspace"
      config: object     # source-specific config
outputs:
  params: object         # filled template parameters
  prompt: string         # rendered prompt text
```

**Asterisk usage:**
- `ContextBuilder` (transformers.go) — implements `framework.Transformer`, reads store + envelope + catalog to build `TemplateParams`
- `PromptFiller` (transformers.go) — implements `framework.Transformer`, fills Go text/template with params
- `BuildParams()` (params.go) — 500 LOC assembling `TemplateParams` from envelope, failure data, prior artifacts, workspace config
- `FillTemplate()` (template.go) — Go text/template rendering
- `ScenarioToCatalog()` (catalog_convert.go) — converts scenario workspace config to knowledge catalog

**Achilles usage:**
- Context assembly is inline in nodes (scanNode reads prior artifact, classifyNode reads scan result). No separate transformer — the marble would formalize what Achilles does implicitly.

**Generalization:** The marble separates "what data goes into the prompt" from "how the prompt is rendered." Data sources are pluggable (store, envelope, file, API). Template engine is Go text/template (already framework-standard). The assembled params are domain-agnostic key-value pairs.

**Origami location:** `marbles/context/` or enhance existing `framework.Transformer` DSL.

---

## 3. `persist`

**Purpose:** Save step artifacts to file and/or SQLite on step completion.

**Interface:**

```yaml
marble: persist
inputs:
  artifact: object       # step output artifact
  step: string           # pipeline step name
  case_id: string        # case identifier
triggers:
  - on: step_complete    # hook trigger
    actions:
      - write_file:
          path: "{{ case_dir }}/{{ step }}.json"
      - sqlite_exec:
          table: string
          columns: [...]
          values: [...]
```

**Asterisk usage:**
- `StoreHooks()` (hooks.go) — returns `framework.HookRegistry` with 5 hooks: `store.recall`, `store.triage`, `store.investigate`, `store.correlate`, `store.review`
- `WriteArtifact()` / `ReadArtifact[T]()` (artifact.go) — JSON file I/O per case directory
- `WritePrompt()` (artifact.go) — saves filled prompt text for debugging
- Each hook calls `apply*Effects()` functions (in runner.go) that update the store

**Achilles usage:**
- Currently no persistence. Future: save findings to a vulnerability database. The marble would provide this capability without Go code.

**Generalization:** The marble is a hook-triggered persistence chain. On step completion, it writes the artifact to file (JSON) and optionally executes SQLite operations via the sqlite adapter. The trigger conditions, file paths, and SQL operations are all YAML-configured.

**Origami location:** `marbles/persist/` using `adapters/sqlite/` for DB operations.

---

## 4. `score`

**Purpose:** Evaluate analysis results against ground truth using a declarative scorecard.

**Interface:**

```yaml
marble: score
inputs:
  results: list          # per-case analysis results
  ground_truth: list     # expected values from scenario
  scorecard: path        # path to scorecard YAML
outputs:
  metrics: list          # computed metric values with pass/fail
  overall: float         # weighted aggregate score
  gaps: list             # evidence gap items (when confidence is low)
```

**Asterisk usage:**
- `computeMetrics()` (metrics.go) — 646 LOC, 21 metric scorers (M1 defect accuracy, M2 symptom accuracy, M3 recall hit rate, ..., M19 overall, M20 variance)
- `ClassifyVerdict()` (evidence_gap.go) — determines Confident/LowConfidence/Inconclusive from convergence score
- `ClusterCases()` (cluster.go) — groups cases by symptom for serial killer detection (M5)
- `ScoreCard` (loaded from `scorecards/asterisk-rca.yaml`) — defines thresholds, weights, formulas

**Achilles usage:**
- `assessNode` computes risk score (0-1) from findings, groups by severity. The scoring is simpler but structurally identical: input data + rules -> numeric scores.

**Generalization:** The marble decouples scoring logic from domain-specific metrics. The ScoreCard YAML defines what to measure and how to aggregate. Individual scorers are registered functions (Go) that the framework invokes. Domain tools define their own scorers; the marble handles loading, evaluation, aggregation, and gap detection.

**Origami location:** Enhance `calibrate/` package with scorer registry + ScoreCard evaluation engine.

---

## 5. `report`

**Purpose:** Generate formatted human-readable output from scored analysis results.

**Interface:**

```yaml
marble: report
inputs:
  results: object        # scored analysis results
  format: string         # "markdown", "terminal", "json"
  template: path         # optional report template
outputs:
  report: string         # formatted report text
```

**Asterisk usage (5 report types):**
- `FormatReport()` (report.go) — calibration report: metrics table with go-pretty
- `RenderRCAReport()` (rca_report.go) — per-RCA report: header, summary, component findings, case details
- `WeaveTranscripts()` (transcript.go) — per-case narrative transcript
- `GenerateBriefing()` (briefing.go) — executive briefing: grouped by RCA
- `BuildCostBill()` / `FormatCostBill()` (tokimeter.go) — token/cost markdown bill

**Achilles usage:**
- `reportNode` — colored terminal output: repo, time, risk score, findings table by severity, fix suggestions.

**Generalization:** The marble provides a template-driven report engine. Report sections are defined in YAML (or Markdown templates). The engine fills templates with structured data and formats for the target medium (terminal colors, Markdown, JSON). Domain tools define their own report templates; the marble handles rendering.

**Origami location:** `marbles/report/` or enhance `format/` package.

---

## 6. `dispatch`

**Purpose:** Route prompts/commands to providers with fallback, retry, and routing instrumentation.

**Interface:**

```yaml
marble: dispatch
inputs:
  intent: string         # what the step needs (e.g. "triage", "scan")
  providers:             # ordered provider list
    - name: string
      config: object
  fallback: string       # fallback strategy: "next", "heuristic", "error"
outputs:
  response: object       # provider response
  routing_log: object    # which provider handled, latency, tokens
```

**Asterisk usage:**
- `BasicAdapter` (adapt/basic.go) — 577 LOC heuristic adapter: keyword-based triage, component detection, store lookups, fallback when LLM unavailable
- `LLMAdapter` (adapt/llm.go) — 269 LOC: prompt dispatch via `dispatch.MuxDispatcher`, stdin template parsing
- `StubAdapter` (adapt/stub.go) — 218 LOC: returns pre-authored artifacts for deterministic calibration
- `RoutingRecorder` (adapt/routing.go) — 214 LOC: wraps any adapter, logs routing decisions to JSON

**Achilles usage:**
- Direct subprocess call to `govulncheck`. No fallback chain, no LLM. But future Achilles phases will add LLM-based assessment — the marble would provide that without Go code.

**Generalization:** The marble manages the provider selection chain. Each provider is registered with a name and config. The marble tries providers in order, with configurable fallback (next provider, heuristic fallback, or error). All routing decisions are logged for observability. The `dispatch` package already exists in Origami; the marble wraps it with YAML configuration.

**Origami location:** Enhance `dispatch/` package with YAML-configured provider chains.

---

## Implementation priority

| Priority | Marble | Rationale |
|----------|--------|-----------|
| 1 | `llm-extract` | Core primitive — every LLM-backed node needs it |
| 2 | `context-builder` | Every prompt needs context assembly |
| 3 | `dispatch` | Provider routing is framework infrastructure |
| 4 | `persist` | Enables stateful pipelines |
| 5 | `score` | Calibration depends on it |
| 6 | `report` | Output formatting is the final layer |

## Achilles cross-validation matrix

| Marble | Asterisk nodes | Achilles nodes | Shared? |
|--------|---------------|----------------|---------|
| `llm-extract` | recall, triage, resolve, investigate, correlate, review, report | scan (govulncheck), classify (deterministic) | Yes — provider is pluggable |
| `context-builder` | All LLM nodes (params + template) | Inline in scan/classify/assess | Yes — formalizes implicit pattern |
| `persist` | recall, triage, investigate, correlate, review (store hooks) | None (future: finding DB) | Yes — optional, hook-triggered |
| `score` | calibrate (M1-M20, scorecard) | assess (risk score) | Yes — scorecard is generic |
| `report` | report (5 formats) | report (terminal) | Yes — template-driven |
| `dispatch` | All LLM nodes (adapter selection + fallback) | scan (subprocess) | Yes — provider is pluggable |
