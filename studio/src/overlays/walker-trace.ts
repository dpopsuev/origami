import type { Edge } from "@xyflow/react";
import type { WalkerTrace } from "./types";

const WALKER_COLORS: Record<string, string> = {
  fire: "#DC143C",
  water: "#007BA7",
  earth: "#0047AB",
  air: "#FFBF00",
  diamond: "#0F52BA",
  lightning: "#DC143C",
};

/**
 * Apply walker trace visualization to edges — animate and color
 * edges based on which walker traversed them.
 */
export function applyWalkerTrace(
  edges: Edge[],
  traces: WalkerTrace[],
  activeWalker?: string
): Edge[] {
  const edgeWalkers = new Map<string, { element: string; persona: string }>();

  for (const trace of traces) {
    if (activeWalker && trace.walkerId !== activeWalker) continue;
    for (let i = 0; i < trace.path.length - 1; i++) {
      const from = trace.path[i].nodeId;
      const to = trace.path[i + 1].nodeId;
      for (const edge of edges) {
        if (edge.source === from && edge.target === to) {
          edgeWalkers.set(edge.id, {
            element: trace.element,
            persona: trace.persona,
          });
        }
      }
    }
  }

  return edges.map((edge) => {
    const walker = edgeWalkers.get(edge.id);
    if (!walker) return edge;

    const color = WALKER_COLORS[walker.element] || "#888";
    return {
      ...edge,
      animated: true,
      style: {
        ...edge.style,
        stroke: color,
        strokeWidth: 3,
      },
      label: `${walker.persona}`,
    };
  });
}
