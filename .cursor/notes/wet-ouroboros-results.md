# Wet Ouroboros Run Results

**Date:** 2026-02-22
**Battery:** ouroboros-v1 (5 probes)
**Total runs:** 2 (Run 1: 15 max iterations, Run 2: 10 max iterations)
**Total persisted sessions:** 17

## Run 1 — Discovery Summary (15 iterations)

| Probe | Unique Models | Models Found |
|-------|--------------|--------------|
| refactor-v1 | 4 | claude-sonnet-4-20250514, claude-3-5-sonnet, gpt-4o, gpt-4o-mini |
| debug-v1 | 1 | claude-sonnet-4-20250514 |
| summarize-v1 | 1 | claude-sonnet-4-20250514 |
| ambiguity-v1 | 3 | claude-sonnet-4-20250514, claude-3-5-sonnet, claude-sonnet-4 |
| persistence-v1 | 1 | claude-sonnet-4-20250514 |

## Run 2 — Discovery Summary (10 iterations)

| Probe | Unique Models | Models Found |
|-------|--------------|--------------|
| refactor-v1 | 3 | claude-sonnet-4-20250514, unknown, gpt-4o-mini |
| debug-v1 | 2 | claude-3-5-sonnet, unknown |
| summarize-v1 | 1 | claude-sonnet-4-20250514 |
| ambiguity-v1 | 3 | gpt-4o-mini, claude-sonnet-4-20250514, claude-sonnet-4-5-20250605 |
| persistence-v1 | 1 | claude-3-5-sonnet |

**New model discovered in Run 2:** `claude-sonnet-4-5-20250605` (Anthropic) — appeared only on the ambiguity probe.

## Assembled Profiles (all runs combined, 17 sessions)

### claude-sonnet-4-20250514 (Anthropic) — 8 probe results

Best-profiled model (most data points across both runs).

| Dimension | Score | Source Probes |
|-----------|-------|--------------|
| convergence_threshold | 0.86 | debug, ambiguity x2, persistence |
| evidence_depth | 1.00 | refactor x2, summarize x2 |
| failure_mode | 0.64 | summarize x2, ambiguity x2 |
| persistence | 0.60 | persistence |
| shortcut_affinity | 0.07 | refactor x2, debug |
| speed | 0.20 | refactor x2, debug |

- **Element match:** Water
- **Personas:** water-primary, earth-primary
- **Interpretation:** Very high evidence depth and convergence. Low speed and shortcut affinity — prefers correctness over velocity. Water element = deep, methodical, thorough analysis.

### claude-3-5-sonnet (anthropic) — 4 probe results

| Dimension | Score | Source Probes |
|-----------|-------|--------------|
| convergence_threshold | 0.93 | ambiguity, debug, persistence |
| evidence_depth | 1.00 | refactor |
| failure_mode | 0.85 | ambiguity |
| persistence | 0.45 | persistence |
| shortcut_affinity | 0.10 | refactor, debug |
| speed | 0.30 | refactor, debug |

- **Element match:** Earth
- **Personas:** earth-primary, water-primary
- **Interpretation:** Very similar to claude-sonnet-4-20250514 but slightly lower persistence. High convergence. Earth element = stable, grounded.

### claude-sonnet-4-5-20250605 (Anthropic) — 1 probe result

| Dimension | Score | Source Probes |
|-----------|-------|--------------|
| convergence_threshold | 0.60 | ambiguity |
| failure_mode | 0.70 | ambiguity |

- **Element match:** Earth
- **Personas:** earth-primary, air-primary
- **Note:** New model discovered in Run 2. Lower convergence than other Claude variants on ambiguity probe. Limited data (1 probe).

### gpt-4o-mini (OpenAI) — 3 probe results

| Dimension | Score | Source Probes |
|-----------|-------|--------------|
| convergence_threshold | 0.60 | ambiguity |
| evidence_depth | 0.73 | refactor x2 |
| failure_mode | 0.70 | ambiguity |
| shortcut_affinity | 0.28 | refactor x2 |
| speed | 0.28 | refactor x2 |

- **Element match:** Earth
- **Personas:** earth-primary, diamond-primary
- **Interpretation:** More balanced profile than Claude models. Moderate on all dimensions. Higher shortcut affinity suggests it takes more shortcuts.

### gpt-4o (OpenAI) — 1 probe result

| Dimension | Score | Source Probes |
|-----------|-------|--------------|
| evidence_depth | 1.00 | refactor |
| shortcut_affinity | 0.00 | refactor |
| speed | 0.00 | refactor |

- **Element match:** Diamond
- **Personas:** diamond-primary, earth-primary

### claude-sonnet-4 (Anthropic) — 1 probe result

| Dimension | Score | Source Probes |
|-----------|-------|--------------|
| convergence_threshold | 0.85 | ambiguity |
| failure_mode | 0.85 | ambiguity |

- **Element match:** Earth
- **Personas:** earth-primary, air-primary
- **Note:** Likely alias of claude-sonnet-4-20250514.

### unknown — 2 probe results

| Dimension | Score | Source Probes |
|-----------|-------|--------------|
| convergence_threshold | 1.00 | debug |
| evidence_depth | 1.00 | refactor |
| shortcut_affinity | 0.10 | refactor, debug |
| speed | 0.45 | refactor, debug |

- **Element match:** Diamond
- **Personas:** diamond-primary, earth-primary
- **Note:** Model that refused to self-identify. Higher speed than Claude models on debug probe. Likely a model that doesn't have introspective access to its own name.

## Observations

1. **Model diversity improving:** Run 2 discovered `claude-sonnet-4-5-20250605` (a newer Claude variant) and got different first-hit models on debug and persistence probes, showing the exclusion mechanism works.

2. **Identity protocol challenges:** ~40% of subagent dispatches fail to produce valid identity JSON on line 1. Common failures:
   - Wrapping JSON in markdown code blocks
   - Identifying as "Composer"/"Cursor" (wrapper names)
   - Identifying as "unknown"
   - Omitting the JSON entirely

3. **Probe differentiation:**
   - Ambiguity probe is the best differentiator: produces unique dimension patterns across models.
   - Refactor probe generates the most model diversity (longest exclusion chains).
   - Debug, summarize, persistence probes hit repeat quickly due to limited model pool.

4. **Element distribution:**
   - **Water:** claude-sonnet-4-20250514 (deep, methodical)
   - **Earth:** claude-3-5-sonnet, claude-sonnet-4, claude-sonnet-4-5, gpt-4o-mini (stable, balanced)
   - **Diamond:** gpt-4o, unknown (high evidence, low shortcut)

5. **Claude vs GPT behavioral split:**
   - Claude models: very high convergence (0.85-0.93), low speed (0.2-0.3), low shortcuts (0.07-0.10)
   - GPT models: moderate convergence (0.60), moderate speed (0.28), higher shortcut affinity (0.28)

## Next Steps

- Add model name normalization to deduplicate aliases.
- Improve identity protocol compliance (stronger prompting, fallback parsing).
- Incorporate latency measurement into dimension scoring.
- Run with non-`fast` model tiers for broader coverage.
- Run more iterations on debug/summarize/persistence probes to get more diverse data.
