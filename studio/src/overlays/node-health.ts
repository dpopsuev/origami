import type { Node } from "@xyflow/react";
import type { NodeMetrics } from "./types";

export interface HealthBadge {
  level: "green" | "yellow" | "red";
  rate: number;
  tooltip: string;
}

function computeHealth(m: NodeMetrics): HealthBadge {
  const total = m.visitCount;
  if (total === 0) {
    return { level: "green", rate: 1, tooltip: "No runs yet" };
  }

  const successRate = 1 - m.errorCount / total;
  if (successRate >= 0.95) {
    return {
      level: "green",
      rate: successRate,
      tooltip: `${(successRate * 100).toFixed(1)}% success (${total} runs)`,
    };
  }
  if (successRate >= 0.8) {
    return {
      level: "yellow",
      rate: successRate,
      tooltip: `${(successRate * 100).toFixed(1)}% success (${m.errorCount} errors / ${total} runs)`,
    };
  }
  return {
    level: "red",
    rate: successRate,
    tooltip: `${(successRate * 100).toFixed(1)}% success (${m.errorCount} errors / ${total} runs)`,
  };
}

const HEALTH_COLORS: Record<string, string> = {
  green: "#22c55e",
  yellow: "#eab308",
  red: "#ef4444",
};

/**
 * Apply node health indicator — colored border ring based on historical success rate.
 */
export function applyNodeHealth(
  nodes: Node[],
  metrics: NodeMetrics[]
): { nodes: Node[]; badges: Map<string, HealthBadge> } {
  const metricMap = new Map(metrics.map((m) => [m.nodeId, m]));
  const badges = new Map<string, HealthBadge>();

  const styledNodes = nodes.map((node) => {
    const m = metricMap.get(node.id);
    if (!m) return node;

    const badge = computeHealth(m);
    badges.set(node.id, badge);

    const color = HEALTH_COLORS[badge.level];
    return {
      ...node,
      style: {
        ...node.style,
        boxShadow: `0 0 0 2px ${color}`,
      },
    };
  });

  return { nodes: styledNodes, badges };
}
