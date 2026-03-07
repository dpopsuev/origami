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

# F6 — Report: Generate Outputs

**Case:** #4  
  
**Step:** report

---

## Task

Generate the final report artifacts: a Jira ticket draft and a regression summary table.

## Approved RCA

| Field | Value |
|-------|-------|
| **RCA message** | root cause from subagent-3 |
| **Defect type** | `pb001` |
| **Convergence** | 0.85 |

**Evidence:**
- ref-1





## Failure context

**Test name:** `OCP-83299 PTP config isolation`  
**Error:**
```
Expected PtpConfig 'test-ptp-config' not to exist in namespace 'openshift-ptp' but it does
```


Defect types:
- pb001: Product Bug — defect in the product code (operator, daemon, proxy, etc.)
- au001: Automation Bug — defect in test code, CI config, or test infrastructure
- en001: Environment Issue — infrastructure/environment issue (node, network, cluster, NTP, etc.)
- fw001: Firmware Issue — defect in firmware or hardware-adjacent code (NIC, FPGA, PHC)
- nd001: No Defect — test is correct, product is correct, flaky/transient/expected behavior
- ti001: To Investigate — insufficient data to classify; needs manual investigation

## Instructions

### 1. Jira ticket draft

Generate a Jira-ready ticket based on the defect type:

**For product bugs (`pb001`):**
- Summary: Clear, searchable title
- Description: Root cause, affected component, reproduction path
- Components: Affected components
- Priority: Based on severity
- Evidence: Links to logs, commits, code

**For other defect types:**
- Adjust the template (automation bugs target test repos, system issues target infra, etc.)

### 2. Regression summary table

Generate a markdown table summarizing all investigated cases:

```markdown
| Case | Test | Defect Type | RCA | Jira | Confidence |
|------|------|-------------|-----|------|------------|
| #N   | name | pb001       | ... | TBD  | 0.85       |
```

## Output format

Save as `jira-draft.json`:

```json
{
  "summary": "Brief Jira title",
  "description": "Full Jira description with root cause, evidence, and fix suggestion",
  "defect_type": "pb001",
  "priority": "High",
  "components": ["component-name"],
  "evidence_refs": ["path/to/evidence"],
  "affected_versions": ["4.21"]
}
```

Also save `regression-report.md` with the regression summary table.
