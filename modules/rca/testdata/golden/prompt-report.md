# F6 — Report: Generate Outputs

**Case:** #7  
**Launch:** launch-42  
**Step:** F6_REPORT

---

## Task

Generate the final report artifacts: a Jira ticket draft and a regression summary table.

## Approved RCA

| Field | Value |
|-------|-------|
| **RCA message** | Holdover timeout changed from 300s to 60s in commit abc1234, causing premature clock class transition to 248. |
| **Defect type** | `pb001` |
| **Convergence** | 0.85 |

**Evidence:**
- linuxptp-daemon:pkg/daemon/config.go:abc1234
- cnf-gotests:test/e2e/ptp_recovery_test.go:TestRecovery



### Cross-version impact

Affected versions: `4.20` `4.21` 


## Failure context

**Test name:** `[T-TSC] PTP Recovery after grandmaster clock switchover`  
**Error:**
```
Expected clock class 6 but got 248 after 300s holdover timeout
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
