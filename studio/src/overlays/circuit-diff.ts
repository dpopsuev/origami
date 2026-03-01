import type { Node, Edge } from "@xyflow/react";
import type { CircuitDiff } from "./types";

/**
 * Apply diff visualization: green for added, red for removed, yellow for modified.
 */
export function applyDiffOverlay(
  nodes: Node[],
  edges: Edge[],
  diff: CircuitDiff
): { nodes: Node[]; edges: Edge[] } {
  const addedNodes = new Set(diff.added.nodes);
  const removedNodes = new Set(diff.removed.nodes);
  const modifiedNodes = new Set(diff.modified.nodes);
  const addedEdges = new Set(diff.added.edges);
  const removedEdges = new Set(diff.removed.edges);
  const modifiedEdges = new Set(diff.modified.edges);

  const styledNodes = nodes.map((node) => {
    let border = node.style?.border;
    let opacity = 1;

    if (addedNodes.has(node.id)) {
      border = "2px solid #22c55e";
    } else if (removedNodes.has(node.id)) {
      border = "2px solid #ef4444";
      opacity = 0.6;
    } else if (modifiedNodes.has(node.id)) {
      border = "2px solid #eab308";
    }

    return { ...node, style: { ...node.style, border, opacity } };
  });

  const styledEdges = edges.map((edge) => {
    let stroke = edge.style?.stroke || "#888";
    let strokeDasharray: string | undefined;

    if (addedEdges.has(edge.id)) {
      stroke = "#22c55e";
    } else if (removedEdges.has(edge.id)) {
      stroke = "#ef4444";
      strokeDasharray = "8 4";
    } else if (modifiedEdges.has(edge.id)) {
      stroke = "#eab308";
    }

    return { ...edge, style: { ...edge.style, stroke, strokeDasharray } };
  });

  return { nodes: styledNodes, edges: styledEdges };
}
