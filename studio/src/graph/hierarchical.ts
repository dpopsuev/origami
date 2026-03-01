/**
 * Hierarchical graph model for subgraph fold/unfold.
 *
 * A zone with multiple nodes can be "folded" into a single composite node.
 * When unfolded, the zone's internal nodes and edges are shown.
 * Cross-level edges (edges that cross a fold boundary) are routed through
 * the composite node's ports.
 */

import type { Node, Edge } from "@xyflow/react";

export interface HierarchyLevel {
  id: string;
  parentId: string | null;
  label: string;
  childNodeIds: string[];
}

export interface FoldState {
  foldedZones: Set<string>;
}

export function createFoldState(): FoldState {
  return { foldedZones: new Set() };
}

export function toggleFold(state: FoldState, zoneId: string): FoldState {
  const next = new Set(state.foldedZones);
  if (next.has(zoneId)) {
    next.delete(zoneId);
  } else {
    next.add(zoneId);
  }
  return { foldedZones: next };
}

export function isFolded(state: FoldState, zoneId: string): boolean {
  return state.foldedZones.has(zoneId);
}

export interface ZoneInfo {
  name: string;
  nodes: string[];
  element?: string;
}

/**
 * Apply fold state to nodes and edges. Folded zones become single composite
 * nodes; their internal edges are hidden; cross-boundary edges are re-routed.
 */
export function applyFoldState(
  nodes: Node[],
  edges: Edge[],
  zones: ZoneInfo[],
  foldState: FoldState
): { nodes: Node[]; edges: Edge[] } {
  const foldedNodeSets = new Map<string, Set<string>>();
  for (const zone of zones) {
    if (foldState.foldedZones.has(zone.name)) {
      foldedNodeSets.set(zone.name, new Set(zone.nodes));
    }
  }

  if (foldedNodeSets.size === 0) {
    return { nodes, edges };
  }

  // Map each folded node to its zone
  const nodeToZone = new Map<string, string>();
  for (const [zoneName, nodeSet] of foldedNodeSets) {
    for (const nodeId of nodeSet) {
      nodeToZone.set(nodeId, zoneName);
    }
  }

  // Filter nodes: remove folded ones, add composite nodes
  const visibleNodes: Node[] = [];
  const compositePositions = new Map<string, { x: number; y: number }>();

  for (const node of nodes) {
    const zone = nodeToZone.get(node.id);
    if (zone) {
      // Accumulate position for composite node centering
      if (!compositePositions.has(zone)) {
        compositePositions.set(zone, { x: 0, y: 0 });
      }
      const pos = compositePositions.get(zone)!;
      pos.x += node.position.x;
      pos.y += node.position.y;
    } else {
      visibleNodes.push(node);
    }
  }

  // Create composite nodes
  for (const zone of zones) {
    if (!foldState.foldedZones.has(zone.name)) continue;
    const pos = compositePositions.get(zone.name);
    const count = zone.nodes.length;
    if (pos && count > 0) {
      visibleNodes.push({
        id: `__zone_${zone.name}`,
        type: "group",
        data: {
          label: `${zone.name} (${count} nodes)`,
          element: zone.element,
          folded: true,
          zoneName: zone.name,
        },
        position: { x: pos.x / count, y: pos.y / count },
        style: {
          background: zone.element
            ? `${elementColor(zone.element)}22`
            : "#33333322",
          border: `2px dashed ${zone.element ? elementColor(zone.element) : "#555"}`,
          borderRadius: "12px",
          padding: "16px",
          minWidth: "120px",
          minHeight: "60px",
        },
      });
    }
  }

  // Re-route edges
  const visibleEdges: Edge[] = [];
  const seenCompositeEdges = new Set<string>();

  for (const edge of edges) {
    const fromZone = nodeToZone.get(edge.source);
    const toZone = nodeToZone.get(edge.target);

    if (fromZone && toZone && fromZone === toZone) {
      // Internal edge — hidden when folded
      continue;
    }

    const source = fromZone ? `__zone_${fromZone}` : edge.source;
    const target = toZone ? `__zone_${toZone}` : edge.target;
    const key = `${source}→${target}`;

    if (seenCompositeEdges.has(key)) continue;
    seenCompositeEdges.add(key);

    visibleEdges.push({
      ...edge,
      id: fromZone || toZone ? `__rerouted_${edge.id}` : edge.id,
      source,
      target,
      style: {
        ...edge.style,
        strokeDasharray: fromZone || toZone ? "5 3" : undefined,
      },
    });
  }

  return { nodes: visibleNodes, edges: visibleEdges };
}

function elementColor(element: string): string {
  const colors: Record<string, string> = {
    fire: "#DC143C",
    water: "#007BA7",
    earth: "#0047AB",
    air: "#FFBF00",
    diamond: "#0F52BA",
    lightning: "#DC143C",
    iron: "#48494B",
  };
  return colors[element] || "#555";
}

/**
 * Build breadcrumb path from root to current depth.
 */
export function buildBreadcrumbs(
  currentZone: string | null,
  zones: ZoneInfo[]
): Array<{ id: string | null; label: string }> {
  const crumbs: Array<{ id: string | null; label: string }> = [
    { id: null, label: "Circuit" },
  ];
  if (currentZone) {
    const zone = zones.find((z) => z.name === currentZone);
    crumbs.push({
      id: currentZone,
      label: zone ? `${zone.name} (${zone.nodes.length} nodes)` : currentZone,
    });
  }
  return crumbs;
}
