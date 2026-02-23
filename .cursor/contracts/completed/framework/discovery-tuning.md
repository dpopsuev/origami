# Contract — discovery-tuning

**Status:** complete  
**Goal:** Discovery identity prompt is fixed (no on-the-fly additions) and compound wrapper names are rejected so RunReport.UniqueModels contains only foundation IDs; prompt compliance on golden set remains ≥ 0.4.  
**Serves:** Framework showcase (weekend side-quest)

## Contract rules

- Follow the **Red-Orange-Green-Yellow-Blue** cycle (see `rules/testing-methodology.mdc`).
- Zero imports from Asterisk domain packages (`calibrate`, `orchestrate`, `origami`). All changes in `pkg/framework/metacal/`, `pkg/framework/known_models.go`, and `internal/metacalmcp/` as needed.
- Wrapper rejection remains a hard guard — no configuration to disable it.

## Context

- **Wet run (2026-02-21):** `.cursor/notes/wet-run-2026-02-21.md` — 10 accepted iterations, 17 total attempts; effectiveness 10/17 ≈ 0.59. Unique models included `cursor-auto`, `cursor-composer`, `cursor-default` because only bare `auto` is rejected today; compound names are accepted.
- **deterministic-agent-identity** (complete): `Session.SubmitResponse` rejects wrapper identities via `framework.IsWrapperName`; prompt compliance scored against golden responses. `IsWrapperName` in `pkg/framework/known_models.go` checks exact match against `KnownWrappers`. `BuildIdentityPrompt()` in `pkg/framework/metacal/discovery.go` defines the identity prompt; main agent added on-the-fly "Reply with ONLY…" instructions during the wet run, which are not part of the tested prompt.

## FSC artifacts

Code only — no FSC artifacts.

## Execution strategy

1. Extend `IsWrapperName` for compound names (tests first, then implementation).
2. Bake "Reply with ONLY…" (and optional line-1 sentence) into `BuildIdentityPrompt`.
3. Run prompt compliance tests; ensure foundation_pct ≥ 0.4; adjust golden expectations only if intended.
4. Validate (green) → Tune (blue) → Validate (green).

## Coverage matrix

| Layer | Applies | Rationale |
|-------|---------|-----------|
| **Unit** | yes | `IsWrapperName` prefix matching, `BuildIdentityPrompt` content, golden response classification |
| **Integration** | no | No new cross-boundary calls; existing `Session.SubmitResponse` tests cover the wrapper guard path |
| **Contract** | no | No API schema changes |
| **E2E** | no | Wet run is optional manual validation, not automated |
| **Concurrency** | no | No shared state changes |
| **Security** | no | Already assessed — no trust boundaries affected |

## Tasks

- [x] Extend `IsWrapperName` to reject `cursor-*`, `composer-*` (and any other known wrapper prefix); add unit tests; ensure session/MCP tests still pass.
- [x] Add one fixed sentence to `BuildIdentityPrompt`: reply with ONLY line 1 JSON, blank line, ```go block; no other text. Optionally add one sentence: first line must be only the JSON object.
- [x] Run `TestPromptCompliance_FoundationRate` and variant comparison; ensure foundation_pct ≥ 0.4.
- [x] Validate (green) — all tests pass, acceptance criteria met.
- [x] Tune (blue) — refactor for quality. No behavior changes.
- [x] Validate (green) — all tests still pass after tuning.

## Acceptance criteria

- **Given** a subagent returns `model_name` equal to a known wrapper or a compound name (e.g. `cursor-auto`), **when** `SubmitResponse` is called, **then** it returns an error and the identity is NOT recorded.
- **Given** the updated prompt, **when** scored against the golden response set, **then** foundation_pct ≥ 0.4.
- RunReport.UniqueModels from a wet run contains no entries whose model_name is a known wrapper or a compound wrapper name (e.g. cursor-auto).

## Security assessment

No trust boundaries affected. Discovery operates within the Cursor IDE session. Probe inputs are synthetic.

## Notes

- 2026-02-21 20:20 — All tasks complete. `IsWrapperName` now rejects compound wrapper names via prefix matching (e.g. `cursor-auto`, `Composer-Agent`, `copilot-chat`). `BuildIdentityPrompt` includes "Reply with ONLY…" sentence (no more on-the-fly additions needed). All tests green: `TestIsWrapperName` (12 cases), `TestPromptCompliance` (foundation_pct 40%, all goldens classified correctly), full `pkg/framework/...` and `internal/metacalmcp/...` suites pass.
