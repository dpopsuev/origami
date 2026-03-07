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

# F1 — Triage: Classify Symptoms

**Case:** #2  
**Launch:** OCP-83297 PTP sync stability ()  
**Step:** triage

---

## Task

Classify the failure symptom from the error output and envelope metadata. No repo access needed — this is a surface-level assessment.

## Failure under investigation

**Test name:** `OCP-83297 PTP sync stability`  
**Status:** open

**Error message:**
```
ptp4l[23456.789]: port 1: FREERUN state, holdover exceeded after 60s (expected 300s)
```


**Log snippet:**
```
2026-02-01T11:30:00Z ptp4l[23456.789]: port 1: FREERUN state, holdover exceeded after 60s
FAIL: Expected clock state to be LOCKED within 300s timeout
```



**Note: Timestamps may originate from different clock planes (executor, test node, SUT). Cross-plane time comparisons may be unreliable.**



## Sibling failures in this launch

| ID | Name | Status |
|----|------|--------|
|  | OCP-83297 PTP sync stability |  |



*No launch attributes available.*


*No linked Jira tickets.*


## Available repos

| Repo | Path | Purpose |
|------|------|---------|
| linuxptp-daemon-operator |  | PTP operator: manages linuxptp-daemon DaemonSet, PtpConfig CRD, clock sync |
| ptp-test-framework |  | E2E test suite for PTP operator: Ginkgo specs, test helpers, fixtures |
| cluster-infra-config |  | CI cluster configuration: job profiles, NTP config, network templates |
| sriov-network-operator |  | SR-IOV network operator: VF allocation, device plugin (NOT PTP-related) |
| cnf-features-deploy |  | CNF deployment manifests and CI profiles: contains job definitions for all telco operators |



## Symptom categories

Classify by **root cause domain** — where does the bug live?

| Category | Meaning | Signal examples | Likely defect type |
|----------|---------|----------------|-------------------|
| `product` | Bug in the product under test (operator, daemon, proxy). Code logic error, wrong state machine transition, incorrect value mapping. | Assertion failures on SUT behavior ("Expected X got Y" on product state), panic/segfault in product code, incorrect sync state, wrong clock class, holdover re-entry timing | pb001 |
| `automation` | Bug in the test framework or test code itself. The product is correct but the test is wrong. | Test harness misconfiguration, wrong test assertion, test setup error, test timeout due to bad polling interval, test code referencing wrong resource | au001 |
| `infra` | Bug in the infrastructure, cluster, or CI environment. Neither product nor test code is at fault. | Node not ready, DNS failure, connection refused, resource quota exceeded, operator not installed, missing CRD, NTP/chrony unreachable, cluster state leftover from prior test | en001 |
| `flake` | Transient, non-reproducible failure. Product and test are both correct but timing or environment conditions caused a one-off failure. | Intermittent timeout, offset variance spike, Eventually timeout on edge-case timing, known unstable test, non-deterministic ordering | nd001 |
| `firmware` | Bug in firmware or hardware-adjacent code (NIC, FPGA, PHC). Not product-level software. | NIC firmware mismatch, FPGA register misconfiguration, PHC clock source error | fw001 |

**Decision guide:**
1. If the error traces to product source code (operator, daemon, proxy) -> `product`
2. If the error is in test assertions, test setup, or test fixtures -> `automation`
3. If the error is from infrastructure, cluster state, or CI environment -> `infra`
4. If the failure is intermittent and non-reproducible, with no clear code or infra fault -> `flake`
5. When uncertain, prefer `product` — in this domain, ~80% of verified bugs are product bugs.

**Key disambiguation — product vs automation:**
- If the error shows a **product behavior discrepancy** (e.g. timeout value changed from 300s to 60s, wrong state transition, incorrect clock class), classify as `product` even if the failure manifests as a test assertion ("Expected X got Y"). The product is doing the wrong thing; the test is correctly catching it.
- Reserve `automation` only for cases where the **test code itself** is wrong: missing cleanup (stale CRDs), wrong assertion target, test setup error, bad polling interval. The product behavior is correct but the test is broken.
- A holdover/sync timeout discrepancy (e.g. "expected 300s" vs "after 60s") is a product configuration change, not a test bug.

**Key disambiguation — infra vs flake:**
- `infra`: the failure has a clear, persistent infrastructure cause (NTP unreachable, node not ready, missing CRD). Re-running would likely fail again unless the infra is fixed.
- `flake`: the failure is transient and non-reproducible — a timing window was missed, a race condition in the environment, or variance caused a threshold violation. Re-running would likely pass. Use `flake` only when there is no persistent root cause.

Defect types:
- pb001: Product Bug — defect in the product code (operator, daemon, proxy, etc.)
- au001: Automation Bug — defect in test code, CI config, or test infrastructure
- en001: Environment Issue — infrastructure/environment issue (node, network, cluster, NTP, etc.)
- fw001: Firmware Issue — defect in firmware or hardware-adjacent code (NIC, FPGA, PHC)
- nd001: No Defect — test is correct, product is correct, flaky/transient/expected behavior
- ti001: To Investigate — insufficient data to classify; needs manual investigation

## Guards

- **G6 (beforesuite-cascade):** Check if multiple failures have identical or near-identical error messages, especially setup/teardown errors. If so, this is likely a **cascade from a shared setup failure** — classify the parent, not each child. Set `cascade_suspected: true`.
- **G7 (eventually-vs-timeout):** If the error contains "Timed out" from Gomega `Eventually` or `Consistently`, classify as `assertion` (expected state was never reached), NOT as `timeout`. Look for "Expected ... to ..." or "polling every ..." patterns.
- **G8 (ordered-spec-poison):** If the failure was aborted due to a prior spec failure in the same ordered container, trace back to the **first failure** and classify that one instead.
- **G9 (skip-count-signal):** If skipped > 40% of total, comment on possible causes (feature gate, setup dependency, ordered container abort).
- **G11 (cascade-error-blindness):** Read the log **chronologically from earliest to latest**. Identify the **first anomaly or error** — this is the most likely root cause.
- **G13 (name-based-guessing):** Do NOT infer root cause from the test name alone. Trace from the **actual error**.
- **G26 (partial-step-conflation):** If this is a TEST-level item with STEP children, identify which specific STEPs failed.
- **Clock skew guard:** Before classifying as `timeout`, check for clock skew. A step that appears to take hours likely has timestamp misalignment, not an actual timeout.

## Instructions

1. Read the error message and log snippet.
2. Classify the symptom using the category table above.
3. Hypothesize a defect type from the taxonomy.
4. Rank candidate repos by relevance to the symptom (using repo purposes).
5. Determine whether repo investigation is needed (`skip_investigation`).
6. Check for cascade patterns, clock skew, and data quality issues.

## Output format

Save as `triage-result.json`:

```json
{
  "symptom_category": "product",
  "severity": "high",
  "defect_type_hypothesis": "pb001",
  "candidate_repos": ["ptp-operator", "cnf-gotests"],
  "skip_investigation": false,
  "clock_skew_suspected": false,
  "cascade_suspected": false,
  "data_quality_notes": ""
}
```
