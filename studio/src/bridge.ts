/**
 * window.__origami bridge — exposes circuit state to Playwright E2E tests.
 * Tests call these functions via page.evaluate(() => window.__origami.snapshot())
 */

export interface OrigamiBridge {
  snapshot: () => CircuitSnapshot;
  nodeCount: () => number;
  edgeCount: () => number;
  selectedNode: () => string | null;
  zoomLevel: () => number;
  foldState: () => Record<string, boolean>;
  yamlContent: () => string;
}

export interface CircuitSnapshot {
  circuit: string;
  nodeCount: number;
  edgeCount: number;
  selectedNode: string | null;
  zoomLevel: number;
  foldState: Record<string, boolean>;
  yamlContent: string;
}

let state = {
  circuit: "",
  nodes: [] as Array<{ id: string }>,
  edges: [] as Array<{ id: string }>,
  selectedNode: null as string | null,
  zoomLevel: 1,
  foldState: {} as Record<string, boolean>,
  yamlContent: "",
};

export function updateBridgeState(update: Partial<typeof state>) {
  state = { ...state, ...update };
}

const bridge: OrigamiBridge = {
  snapshot: () => ({
    circuit: state.circuit,
    nodeCount: state.nodes.length,
    edgeCount: state.edges.length,
    selectedNode: state.selectedNode,
    zoomLevel: state.zoomLevel,
    foldState: { ...state.foldState },
    yamlContent: state.yamlContent,
  }),
  nodeCount: () => state.nodes.length,
  edgeCount: () => state.edges.length,
  selectedNode: () => state.selectedNode,
  zoomLevel: () => state.zoomLevel,
  foldState: () => ({ ...state.foldState }),
  yamlContent: () => state.yamlContent,
};

declare global {
  interface Window {
    __origami: OrigamiBridge;
  }
}

export function installBridge() {
  window.__origami = bridge;
}
