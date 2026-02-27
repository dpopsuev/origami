/**
 * window.__origami bridge — exposes pipeline state to Playwright E2E tests.
 * Tests call these functions via page.evaluate(() => window.__origami.snapshot())
 */

export interface OrigamiBridge {
  snapshot: () => PipelineSnapshot;
  nodeCount: () => number;
  edgeCount: () => number;
  selectedNode: () => string | null;
  zoomLevel: () => number;
  foldState: () => Record<string, boolean>;
  yamlContent: () => string;
}

export interface PipelineSnapshot {
  pipeline: string;
  nodeCount: number;
  edgeCount: number;
  selectedNode: string | null;
  zoomLevel: number;
  foldState: Record<string, boolean>;
  yamlContent: string;
}

let state = {
  pipeline: "",
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
    pipeline: state.pipeline,
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
