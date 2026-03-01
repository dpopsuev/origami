export type TestStatus = "pending" | "running" | "passed" | "failed" | "skipped";

export interface NodeTestResult {
  nodeId: string;
  status: TestStatus;
  durationMs?: number;
  error?: string;
  stubOutput?: string;
}

export interface TestRunResult {
  circuit: string;
  startedAt: string;
  finishedAt?: string;
  nodes: NodeTestResult[];
  edgesEvaluated: number;
  edgesTrue: number;
  edgesFalse: number;
  passed: boolean;
}

/**
 * Node border styles for testing mode overlay.
 */
const TEST_COLORS: Record<TestStatus, string> = {
  pending: "#6b7280",
  running: "#3b82f6",
  passed: "#22c55e",
  failed: "#ef4444",
  skipped: "#9ca3af",
};

/**
 * Compute node styles for test results overlay.
 */
export function testResultNodeStyles(
  results: NodeTestResult[]
): Map<string, { border: string; opacity: number }> {
  const styles = new Map<string, { border: string; opacity: number }>();

  for (const r of results) {
    const color = TEST_COLORS[r.status];
    styles.set(r.nodeId, {
      border: `2px solid ${color}`,
      opacity: r.status === "skipped" ? 0.5 : 1,
    });
  }

  return styles;
}

/**
 * Summarize test results for display.
 */
export function testSummary(result: TestRunResult): string {
  const passed = result.nodes.filter((n) => n.status === "passed").length;
  const failed = result.nodes.filter((n) => n.status === "failed").length;
  const skipped = result.nodes.filter((n) => n.status === "skipped").length;
  const total = result.nodes.length;

  return `${passed}/${total} passed${failed > 0 ? `, ${failed} failed` : ""}${
    skipped > 0 ? `, ${skipped} skipped` : ""
  } — edges: ${result.edgesTrue}/${result.edgesEvaluated} true`;
}
