import type { Node } from "@xyflow/react";
import type { NodeMetrics } from "./types";

/**
 * Apply heatmap coloring to nodes based on visit frequency or duration.
 */
export function applyHeatmap(
  nodes: Node[],
  metrics: NodeMetrics[],
  mode: "frequency" | "duration" | "errors" = "frequency"
): Node[] {
  const metricMap = new Map(metrics.map((m) => [m.nodeId, m]));

  let maxVal = 0;
  for (const m of metrics) {
    const val = mode === "frequency" ? m.visitCount : mode === "duration" ? m.avgDurationMs : m.errorCount;
    if (val > maxVal) maxVal = val;
  }

  if (maxVal === 0) return nodes;

  return nodes.map((node) => {
    const m = metricMap.get(node.id);
    if (!m) return node;

    const val = mode === "frequency" ? m.visitCount : mode === "duration" ? m.avgDurationMs : m.errorCount;
    const intensity = val / maxVal;

    const color = mode === "errors"
      ? `rgba(220, 20, 60, ${0.2 + intensity * 0.8})`
      : `rgba(0, 123, 167, ${0.2 + intensity * 0.8})`;

    return {
      ...node,
      style: {
        ...node.style,
        background: color,
        boxShadow: intensity > 0.7 ? `0 0 12px ${color}` : undefined,
      },
    };
  });
}
