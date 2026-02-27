import type { Node, Edge } from "@xyflow/react";

type Direction = "forward" | "backward";

/**
 * Build adjacency maps for forward and backward traversal.
 */
function buildAdjacency(edges: Edge[]): {
  forward: Map<string, string[]>;
  backward: Map<string, string[]>;
} {
  const forward = new Map<string, string[]>();
  const backward = new Map<string, string[]>();

  for (const e of edges) {
    if (!forward.has(e.source)) forward.set(e.source, []);
    forward.get(e.source)!.push(e.target);

    if (!backward.has(e.target)) backward.set(e.target, []);
    backward.get(e.target)!.push(e.source);
  }

  return { forward, backward };
}

/**
 * Collect all reachable node IDs from a starting node via BFS.
 */
function reachable(
  start: string,
  adjacency: Map<string, string[]>
): Set<string> {
  const visited = new Set<string>();
  const queue = [start];

  while (queue.length > 0) {
    const current = queue.shift()!;
    if (visited.has(current)) continue;
    visited.add(current);

    const neighbors = adjacency.get(current) || [];
    for (const n of neighbors) {
      if (!visited.has(n)) queue.push(n);
    }
  }

  return visited;
}

/**
 * Highlight all paths reachable from (or leading to) a selected node.
 * Non-reachable nodes are dimmed.
 */
export function applyPathExplorer(
  nodes: Node[],
  edges: Edge[],
  selectedNodeId: string,
  direction: Direction = "forward"
): { nodes: Node[]; edges: Edge[] } {
  const { forward, backward } = buildAdjacency(edges);
  const adj = direction === "forward" ? forward : backward;
  const reachableSet = reachable(selectedNodeId, adj);
  reachableSet.add(selectedNodeId);

  const reachableEdges = new Set<string>();
  for (const e of edges) {
    const src = direction === "forward" ? e.source : e.target;
    const tgt = direction === "forward" ? e.target : e.source;
    if (reachableSet.has(src) && reachableSet.has(tgt)) {
      reachableEdges.add(e.id);
    }
  }

  const styledNodes = nodes.map((node) => ({
    ...node,
    style: {
      ...node.style,
      opacity: reachableSet.has(node.id) ? 1 : 0.2,
      ...(node.id === selectedNodeId
        ? { boxShadow: "0 0 0 3px #3b82f6" }
        : {}),
    },
  }));

  const styledEdges = edges.map((edge) => ({
    ...edge,
    animated: reachableEdges.has(edge.id),
    style: {
      ...edge.style,
      stroke: reachableEdges.has(edge.id) ? "#3b82f6" : "#333",
      strokeWidth: reachableEdges.has(edge.id) ? 2.5 : 1,
      opacity: reachableEdges.has(edge.id) ? 1 : 0.15,
    },
  }));

  return { nodes: styledNodes, edges: styledEdges };
}
