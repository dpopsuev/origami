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

# F5 — Review: Present Findings

**Case:** #2  
  
**Step:** F5_REVIEW

---

## Human Review Gate

This step presents the investigation findings for your review. **No write to RP happens until you approve.**

## Summary

**Test name:** `OCP-83297 PTP sync stability`

### Investigation result

| Field | Value |
|-------|-------|
| **RCA message** | root cause from subagent-0 |
| **Defect type** | `pb001` |
| **Convergence score** | 0.85 |

**Evidence:**
- ref-1





### Triage classification

- Category: `product`
- Defect hypothesis: `pb001`




### Correlation result

- Not a duplicate (confidence: 0.1)
- Reasoning: 



## Decision

Choose one of the following:

### ✅ Approve
The RCA is correct. Proceed to report generation (F6).

### 🔄 Reassess
The RCA needs rework. Specify where to loop back:
- `F1_TRIAGE` — wrong symptom classification
- `F2_RESOLVE` — wrong repo chosen
- `F3_INVESTIGATE` — missed something in the repo

### ❌ Overturn
The RCA is wrong. Provide the correct answer.

## Output format

Save as `review-decision.json`:

```json
{
  "decision": "approve",
  "human_override": null,
  "loop_target": ""
}
```

For reassess:
```json
{
  "decision": "reassess",
  "human_override": null,
  "loop_target": "F2_RESOLVE"
}
```

For overturn:
```json
{
  "decision": "overturn",
  "human_override": {
    "defect_type": "au001",
    "rca_message": "The actual root cause is..."
  },
  "loop_target": ""
}
```
