# Contract — origami-papercup-v2-hardening

**Status:** complete  
**Goal:** Align Origami's developer-facing documentation with Papercup v2 choreography so new skills are built on the v2 pattern from the start.  
**Serves:** Framework Maturity  
**Companion:** `asterisk-papercup-v2-hardening` (Asterisk repo — server-side embedding + skill rewrite)

## Contract rules

- The Cursor Skill guide MUST describe the v2 supervisor/worker pattern, not v1 orchestration.
- The guide MUST reference server-generated `worker_prompt` and inline `prompt_content`.
- Changes are documentation-only; no Origami Go code changes in this contract.

## Context

- `docs/cursor-skill-guide.md` — Developer guide for building Cursor Skills from Origami pipelines. Agent Bus Protocol section (lines 89-118) previously described v1 orchestration.
- `rules/domain/agent-bus.mdc` — Papercup v2 protocol specification (already v2).
- `contracts/active/papercup-protocol-maturity.md` — Parent contract for the full Papercup evolution.

## Tasks

- [x] **P2.2** Rewrite `docs/cursor-skill-guide.md` "Agent Bus Protocol" section: flipped responsibility table to v2, replaced "Parallel mode" with supervisor/worker pattern, documented server-generated `worker_prompt`, inline `prompt_content`, and `peakPullers` capacity detection.
- [x] **P2.5** Tune (blue) — reviewed guide for clarity; v2 pattern described consistently.

## Acceptance criteria

```gherkin
Given the Cursor Skill developer guide
When a developer reads the Agent Bus Protocol section
Then it describes the v2 supervisor/worker pattern
  And mentions server-generated worker_prompt
  And mentions inline prompt_content
  And does NOT describe v1 batch-pull orchestration
```

## Notes

2026-02-24 — Split from combined `papercup-v2-hardening` contract. The Asterisk half contains server-side Go code changes (WorkerPrompt, inline prompt, gate messages, mode tracking, 11 tests) and the skill rewrite. This Origami half contains the documentation alignment to ensure the framework's developer guide teaches the correct v2 pattern.
