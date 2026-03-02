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

# F2 — Resolve: Select Repos and Scope

**Case:** #7  
**Launch:** OCP-83300 PTP config cleanup ()  
**Step:** F2_RESOLVE

---

## Task

Given the triage classification and the available repos, select which repo(s) to investigate and narrow the focus to specific paths/modules.

## Triage result (from F1)

| Field | Value |
|-------|-------|
| Symptom category | product |
| Severity | high |
| Defect type hypothesis | pb001 |
| Candidate repos | `test-repo` |
| Skip investigation | false |






## Prior investigation (loop retry)

Previous investigation converged at **0.85** with defect type `pb001`:

> root cause from subagent-0

The convergence was too low. Select a different repo or broader scope for the retry.


## Failure context

**Test name:** `OCP-83300 PTP config cleanup`  
**Error message:**
```
Expected PtpConfig 'bc-test-config' not to exist in namespace 'openshift-ptp' but it does
```




*No launch attributes available.*


*No linked Jira tickets.*


## Available repos

| Repo | Path | Purpose | Branch |
|------|------|---------|--------|
| linuxptp-daemon-operator |  | PTP operator: manages linuxptp-daemon DaemonSet, PtpConfig CRD, clock sync | release-4.21 |
| ptp-test-framework |  | E2E test suite for PTP operator: Ginkgo specs, test helpers, fixtures | main |
| cluster-infra-config |  | CI cluster configuration: job profiles, NTP config, network templates | main |
| sriov-network-operator |  | SR-IOV network operator: VF allocation, device plugin (NOT PTP-related) | release-4.21 |
| cnf-features-deploy |  | CNF deployment manifests and CI profiles: contains job definitions for all telco operators | master |



## Guards

- **G4 (empty-envelope-fields):** If a field is unavailable or empty, do not assume a value. State what is missing and how it limits the analysis.
- **G18 (env-only-failure):** Consider whether the failure could be **environment-only** — code is correct but the runtime environment differs. If `Env.*` attributes show an unexpected version, include the CI config repo.
- **G28 (config-vs-code):** If the triage symptom is `config` or `infra`, prioritize the CI config repo over code repos.

## Instructions

1. Using the triage result and repo purposes, select the **single most relevant repo** for the root cause.
2. Only add a second repo if the error **clearly spans two components** (e.g. test code calls product API incorrectly — need both). In most cases, one repo is sufficient.
3. For each repo, specify focus paths (directories/files to look at) and why.
4. If multiple repos are needed, describe a cross-reference strategy.
5. If this is a loop retry, select a **different** repo or broader scope than the previous attempt.

**Repo selection by defect type:**

| Triage hypothesis | Preferred repo type | Reasoning |
|---|---|---|
| Product bug | Product / operator repo | The root cause lives in the product code, not in the test that revealed it. |
| Automation bug | Test / framework repo | The root cause is in test logic, assertions, or setup code. |
| Environment issue | CI config / infra repo | The root cause is in environment configuration. |

**CRITICAL:** Test frameworks contain assertions that **reveal** symptoms. When the hypothesis is a product bug, the test framework shows **what failed** but not **why** — the root cause is in the product repo where the buggy code lives. Use the `Purpose` column in the Available repos table to identify which repos contain product code vs test code.

**Precision over breadth:** Selecting too many repos dilutes investigation focus. A wrong repo wastes an investigation step. When in doubt, pick the single repo whose purpose most closely matches the triage hypothesis and defect type.

## Output format

Save as `resolve-result.json`:

```json
{
  "selected_repos": [
    {
      "name": "ptp-operator",
      "path": "/path/to/ptp-operator",
      "focus_paths": ["pkg/daemon/", "api/v1/"],
      "branch": "release-4.21",
      "reason": "Triage indicates product bug in PTP sync; daemon code is the likely location."
    }
  ],
  "cross_ref_strategy": "Check test assertion in cnf-gotests, then verify SUT behavior in ptp-operator."
}
```
