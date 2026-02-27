import type { Node } from "@xyflow/react";

export type BulkAction =
  | "move-to-zone"
  | "delete"
  | "set-breakpoint"
  | "clear-breakpoint"
  | "export-subgraph"
  | "apply-tag";

export interface BulkActionDef {
  id: BulkAction;
  label: string;
  icon: string;
  destructive: boolean;
}

export const BULK_ACTIONS: BulkActionDef[] = [
  { id: "move-to-zone", label: "Move to Zone", icon: "📦", destructive: false },
  { id: "delete", label: "Delete", icon: "🗑", destructive: true },
  { id: "set-breakpoint", label: "Set Breakpoint", icon: "🔴", destructive: false },
  { id: "clear-breakpoint", label: "Clear Breakpoint", icon: "⚪", destructive: false },
  { id: "export-subgraph", label: "Export Subgraph", icon: "📤", destructive: false },
  { id: "apply-tag", label: "Apply Tag", icon: "🏷", destructive: false },
];

export interface LassoRect {
  x: number;
  y: number;
  width: number;
  height: number;
}

/**
 * Determine which nodes fall within the lasso selection rectangle.
 */
export function nodesInLasso(nodes: Node[], rect: LassoRect): string[] {
  const result: string[] = [];

  for (const node of nodes) {
    const nx = node.position.x;
    const ny = node.position.y;

    if (
      nx >= rect.x &&
      nx <= rect.x + rect.width &&
      ny >= rect.y &&
      ny <= rect.y + rect.height
    ) {
      result.push(node.id);
    }
  }

  return result;
}

/**
 * Toggle a node ID in the selection set (shift-click behavior).
 */
export function toggleSelection(
  selection: Set<string>,
  nodeId: string
): Set<string> {
  const next = new Set(selection);
  if (next.has(nodeId)) {
    next.delete(nodeId);
  } else {
    next.add(nodeId);
  }
  return next;
}

/**
 * Apply selection highlight styling to nodes.
 */
export function applySelectionStyle(
  nodes: Node[],
  selected: Set<string>
): Node[] {
  return nodes.map((node) => ({
    ...node,
    style: {
      ...node.style,
      ...(selected.has(node.id)
        ? { boxShadow: "0 0 0 2px #3b82f6, 0 0 8px rgba(59,130,246,0.3)" }
        : {}),
    },
  }));
}
