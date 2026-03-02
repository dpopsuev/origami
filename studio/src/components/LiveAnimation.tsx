import { useEffect, useRef, useState } from "react";
import type { Node, Edge } from "@xyflow/react";

interface LiveEvent {
  type: string;
  node?: string;
  edge?: string;
  walker?: string;
}

const ELEMENT_COLORS: Record<string, string> = {
  fire: "#DC143C",
  water: "#007BA7",
  earth: "#0047AB",
  air: "#FFBF00",
  diamond: "#0F52BA",
  lightning: "#DC143C",
};

/**
 * Hook that subscribes to SSE events and returns styled nodes/edges
 * with live animation (active node glow, edge pulse on transition).
 */
export function useLiveAnimation(
  runId: string | null,
  baseNodes: Node[],
  baseEdges: Edge[]
): { nodes: Node[]; edges: Edge[]; activeWalker: string | null } {
  const [activeNodes, setActiveNodes] = useState<Map<string, string>>(new Map());
  const [visitedNodes, setVisitedNodes] = useState<Set<string>>(new Set());
  const [lastTransition, setLastTransition] = useState<string | null>(null);
  const [activeWalker, setActiveWalker] = useState<string | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    if (!runId) return;

    setActiveNodes(new Map());
    setVisitedNodes(new Set());
    setLastTransition(null);

    const es = new EventSource(`/api/runs/${runId}/events/stream`);
    eventSourceRef.current = es;

    es.onmessage = (msg) => {
      try {
        const evt: LiveEvent = JSON.parse(msg.data);

        if (evt.type === "node_enter" && evt.node) {
          setActiveNodes((prev) => {
            const next = new Map(prev);
            next.set(evt.node!, evt.walker || "default");
            return next;
          });
          setActiveWalker(evt.walker || null);
        }

        if (evt.type === "node_exit" && evt.node) {
          setActiveNodes((prev) => {
            const next = new Map(prev);
            next.delete(evt.node!);
            return next;
          });
          setVisitedNodes((prev) => new Set(prev).add(evt.node!));
        }

        if (evt.type === "transition" && evt.edge) {
          setLastTransition(evt.edge);
          setTimeout(() => setLastTransition(null), 1500);
        }
      } catch {
        // skip malformed events
      }
    };

    return () => {
      es.close();
      eventSourceRef.current = null;
    };
  }, [runId]);

  const styledNodes = baseNodes.map((node) => {
    const walkerEl = activeNodes.get(node.id);
    const isActive = !!walkerEl;
    const isVisited = visitedNodes.has(node.id);

    if (isActive) {
      const color = ELEMENT_COLORS[walkerEl] || "#22c55e";
      return {
        ...node,
        style: {
          ...node.style,
          boxShadow: `0 0 12px ${color}, 0 0 4px ${color}`,
          transition: "box-shadow 0.3s ease",
        },
      };
    }

    if (isVisited) {
      return {
        ...node,
        style: {
          ...node.style,
          opacity: 0.7,
          boxShadow: "0 0 0 1px rgba(34, 197, 94, 0.3)",
        },
      };
    }

    return node;
  });

  const styledEdges = baseEdges.map((edge) => {
    if (edge.id === lastTransition) {
      return {
        ...edge,
        animated: true,
        style: {
          ...edge.style,
          stroke: "#3b82f6",
          strokeWidth: 3,
        },
      };
    }
    return edge;
  });

  return { nodes: styledNodes, edges: styledEdges, activeWalker };
}
