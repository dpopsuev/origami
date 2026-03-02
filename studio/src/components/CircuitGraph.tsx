import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
  type OnSelectionChangeParams,
  BackgroundVariant,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import dagre from "dagre";
import { updateBridgeState } from "../bridge";

const ELEMENT_COLORS: Record<string, string> = {
  fire: "#DC143C",
  water: "#007BA7",
  earth: "#0047AB",
  air: "#FFBF00",
  diamond: "#0F52BA",
  lightning: "#DC143C",
};

interface CircuitNode {
  name: string;
  element?: string;
  family?: string;
}

interface CircuitEdge {
  id: string;
  from: string;
  to: string;
  name?: string;
  when?: string;
}

interface CircuitDef {
  circuit: string;
  nodes: CircuitNode[];
  edges: CircuitEdge[];
  start?: string;
  done?: string;
}

function layoutGraph(
  nodes: Node[],
  edges: Edge[],
  direction = "TB"
): Node[] {
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: direction, nodesep: 50, ranksep: 80 });

  for (const node of nodes) {
    g.setNode(node.id, { width: 180, height: 50 });
  }
  for (const edge of edges) {
    g.setEdge(edge.source, edge.target);
  }

  dagre.layout(g);

  return nodes.map((node) => {
    const pos = g.node(node.id);
    return {
      ...node,
      position: { x: pos.x - 90, y: pos.y - 25 },
    };
  });
}

function circuitToFlow(def: CircuitDef): { nodes: Node[]; edges: Edge[] } {
  const flowNodes: Node[] = def.nodes.map((n) => ({
    id: n.name,
    data: {
      label: n.name,
      element: n.element,
      family: n.family,
    },
    position: { x: 0, y: 0 },
    style: {
      background: n.element ? ELEMENT_COLORS[n.element] || "#333" : "#333",
      color: "#fff",
      border: n.name === def.start ? "2px solid #fff" : "1px solid #555",
      borderRadius: "8px",
      padding: "8px 16px",
      fontSize: "13px",
      fontWeight: n.name === def.start ? "bold" : "normal",
    },
  }));

  const flowEdges: Edge[] = def.edges.map((e) => ({
    id: e.id,
    source: e.from,
    target: e.to,
    label: e.name || "",
    animated: false,
    style: { stroke: "#888" },
  }));

  return { nodes: layoutGraph(flowNodes, flowEdges), edges: flowEdges };
}

const DEMO_PIPELINE: CircuitDef = {
  circuit: "demo",
  nodes: [
    { name: "ingest", element: "fire", family: "ingest" },
    { name: "triage", element: "water", family: "classify" },
    { name: "analyze", element: "earth", family: "analyze" },
    { name: "review", element: "diamond", family: "review" },
    { name: "DONE" },
  ],
  edges: [
    { id: "E1", from: "ingest", to: "triage", name: "ingested", when: "true" },
    { id: "E2", from: "triage", to: "analyze", name: "triaged", when: "output.confidence > 0.5" },
    { id: "E3", from: "analyze", to: "review", name: "analyzed", when: "true" },
    { id: "E4", from: "review", to: "DONE", name: "done", when: "true" },
  ],
  start: "ingest",
  done: "DONE",
};

export function CircuitGraph() {
  const [circuit, setCircuit] = useState<CircuitDef>(DEMO_PIPELINE);

  const { nodes: initialNodes, edges: initialEdges } = useMemo(
    () => circuitToFlow(circuit),
    [circuit]
  );

  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  useEffect(() => {
    const { nodes: n, edges: e } = circuitToFlow(circuit);
    setNodes(n);
    setEdges(e);
  }, [circuit, setNodes, setEdges]);

  const onSelectionChange = useCallback(
    ({ nodes: selected }: OnSelectionChangeParams) => {
      const sel = selected.length > 0 ? selected[0].id : null;
      updateBridgeState({ selectedNode: sel });
    },
    []
  );

  useEffect(() => {
    updateBridgeState({
      circuit: circuit.circuit,
      nodes: nodes.map((n) => ({ id: n.id })),
      edges: edges.map((e) => ({ id: e.id })),
    });
  }, [nodes, edges, circuit]);

  useEffect(() => {
    const handler = async () => {
      try {
        const resp = await fetch("/api/circuits");
        if (resp.ok) {
          const data = await resp.json();
          if (data.circuit) {
            setCircuit(data);
          }
        }
      } catch {
        // API not available, keep demo circuit
      }
    };
    handler();
  }, []);

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      onSelectionChange={onSelectionChange}
      fitView
      minZoom={0.1}
      maxZoom={4}
    >
      <Background variant={BackgroundVariant.Dots} gap={16} size={1} color="#333" />
      <Controls />
      <MiniMap
        nodeColor={(n) => {
          const el = n.data?.element as string | undefined;
          return el ? ELEMENT_COLORS[el] || "#555" : "#555";
        }}
        maskColor="rgba(0,0,0,0.7)"
      />
    </ReactFlow>
  );
}
