import type { Node } from "@xyflow/react";
import type { NodeMetrics } from "./types";

export type ZoomLevel = "low" | "medium" | "high";

/**
 * Determine zoom level from React Flow viewport zoom value.
 */
export function zoomLevel(zoom: number): ZoomLevel {
  if (zoom < 0.5) return "low";
  if (zoom < 1.5) return "medium";
  return "high";
}

interface SemanticLabel {
  primary: string;
  secondary?: string;
  detail?: string;
}

/**
 * Build semantic labels for a node at a given zoom level.
 *
 * Low zoom: name + health badge only.
 * Medium zoom: name + type/zone + last run status.
 * High zoom: full config, recent artifacts, metrics.
 */
export function semanticLabel(
  nodeId: string,
  nodeData: Record<string, unknown>,
  metrics: NodeMetrics | undefined,
  level: ZoomLevel
): SemanticLabel {
  const name = (nodeData.label as string) || nodeId;

  if (level === "low") {
    return { primary: name };
  }

  const element = (nodeData.element as string) || "";
  const family = (nodeData.family as string) || "";
  const secondary = [element, family].filter(Boolean).join(" · ");

  if (level === "medium") {
    const status = metrics
      ? `${metrics.visitCount} runs · ${metrics.avgDurationMs}ms avg`
      : "";
    return { primary: name, secondary, detail: status };
  }

  const detail = metrics
    ? [
        `Visits: ${metrics.visitCount}`,
        `Avg: ${metrics.avgDurationMs}ms`,
        `Errors: ${metrics.errorCount}`,
        metrics.walkers.length > 0
          ? `Walkers: ${metrics.walkers.join(", ")}`
          : "",
      ]
        .filter(Boolean)
        .join("\n")
    : "";

  return { primary: name, secondary, detail };
}

/**
 * Apply semantic zoom styling — adjust node dimensions and font sizes.
 */
export function applySemanticZoom(nodes: Node[], level: ZoomLevel): Node[] {
  const fontSize = level === "low" ? "10px" : level === "medium" ? "13px" : "14px";
  const padding = level === "low" ? "4px 8px" : level === "medium" ? "8px 16px" : "12px 20px";

  return nodes.map((node) => ({
    ...node,
    style: {
      ...node.style,
      fontSize,
      padding,
    },
  }));
}
