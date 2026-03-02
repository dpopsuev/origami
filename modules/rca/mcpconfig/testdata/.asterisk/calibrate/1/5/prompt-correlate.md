> **CALIBRATION MODE — BLIND EVALUATION**
>
> You are participating in a calibration run. Your responses at each circuit
> step will be **scored against known ground truth** using 20 metrics including
> defect type accuracy, component identification, evidence quality, circuit
> path efficiency, and semantic relevance.
>
> **Rules:**
> 1. Respond ONLY based on the information provided in this prompt.
> 2. Do NOT read scenario definition files, ground truth files, expected
>    results, or any calibration/test code in the repository. This includes
>    any file under `internal/calibrate/scenarios/`, any `*_test.go` file,
>    and the `.cursor/contracts/` directory.
> 3. Do NOT look at previous artifact files for other cases unless
>    explicitly referenced in the prompt context.
> 4. Treat each step independently — base your output solely on the
>    provided context for THIS step.
>
> Violating these rules contaminates the calibration signal.

# F4 — Correlate: Match Cases

**Case:** #5  
  
**Step:** F4_CORRELATE

---

## Task

Determine whether this case's root cause matches another case in the same launch, circuit, or suite. Detect "serial killers" (same root cause spanning multiple cases or versions).

## Investigation result (from F3)

| Field | Value |
|-------|-------|
| RCA message | root cause from subagent-1 |
| Defect type | pb001 |
| Convergence | 0.85 |
| Evidence | `ref-1`  |




## Sibling failures in this launch

| ID | Name | Status |
|----|------|--------|
| 0 | OCP-83300 PTP config cleanup |  |



## Guards

- **G23 (false-dedup):** Name similarity is not cause similarity. Before linking two cases to the same RCA, verify: (1) actual error messages match, (2) failure code path is the same, (3) environment context is comparable.
- **G24 (version-crossing-false-equiv):** Same test failing in different versions may have different root causes. Compare actual error details and environment.
- **G25 (shared-setup-misattribution):** If multiple cases share identical error messages pointing to setup, link them to **one RCA for the setup failure**.

## Instructions

1. Compare the current case's RCA against sibling failures and prior RCAs for this symptom.
2. Check if the **actual error messages** match (not just test names).
3. Check for cross-version patterns: same symptom across 4.20, 4.21, 4.22 with the same RCA = "serial killer".
4. If duplicate, specify the linked RCA ID.

## Output format

Save as `correlate-result.json`:

```json
{
  "is_duplicate": false,
  "linked_rca_id": 0,
  "confidence": 0.3,
  "reasoning": "Different error patterns despite similar test names.",
  "cross_version_match": false,
  "affected_versions": []
}
```
