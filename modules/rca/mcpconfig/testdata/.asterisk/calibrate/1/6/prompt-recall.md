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

# F0 — Recall: Judge Similarity

**Case:** #6  
**Test:** OCP-83297 PTP sync stability  
**Step:** F0_RECALL

---

## Task

Determine whether this failure has been seen before by comparing it against prior symptom and RCA data.

## Failure under investigation

**Test name:** `OCP-83297 PTP sync stability`  
**Error message:**
```
ptp4l[45678.901]: port 1: FREERUN state, holdover exceeded after 60s (expected 300s)
```

**Log snippet:**
```
2026-02-05T11:00:00Z ptp4l[45678.901]: port 1: FREERUN state, holdover exceeded after 60s
FAIL: Expected clock state to be LOCKED
```



## Known symptom

| Field | Value |
|-------|-------|
| Name | OCP-83297 PTP sync stability |
| Status | active |
| Occurrences | 1 |
| First seen | 2026-03-02T16:02:26Z |
| Last seen | 2026-03-02T16:02:26Z |






## All known RCAs in this run

These RCAs were discovered from other cases in the current calibration run. If the current failure's error pattern matches any of these, set `match: true` with the matching RCA ID and high confidence.

| RCA ID | Component | Defect Type | Summary |
|--------|-----------|-------------|---------|
| #1 | test-component | pb001 | root cause from subagent-3 |
| #2 | test-component | pb001 | root cause from subagent-2 |



## Guards



- **G5 (stale-recall-match):** When judging similarity to a prior RCA, compare not only the error pattern but also the environment context (OCP version, operator version, cluster). A test can fail for different reasons in different versions. If the environment differs significantly, lower your match confidence.

## Instructions

1. Compare the current failure's error pattern against the known symptom and prior RCAs above.
2. Consider whether the **environment context** (versions, cluster) matches — same test can fail differently across versions.
3. If a prior RCA's symptom was marked as `dormant` or `resolved` and this failure matches, flag `is_regression: true`.
4. Produce the output JSON below.

## Output format

Save as `recall-result.json`:

```json
{
  "match": true,
  "prior_rca_id": 42,
  "symptom_id": 7,
  "confidence": 0.85,
  "reasoning": "Same error pattern as RCA #42: ...",
  "is_regression": false
}
```

- `match`: true if a prior RCA likely explains this failure.
- `prior_rca_id`: the RCA ID if matched, 0 otherwise.
- `symptom_id`: the symptom ID if matched, 0 otherwise.
- `confidence`: 0.0–1.0 (>= 0.8 = high-confidence hit; 0.4–0.8 = uncertain; < 0.4 = miss).
- `reasoning`: brief explanation of match or mismatch.
- `is_regression`: true if this is a known-resolved or dormant symptom reappearing.
