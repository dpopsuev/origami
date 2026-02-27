/**
 * Overlay types for diagnostic and analytics visualizations.
 */

export type OverlayType =
  | "heatmap"
  | "walker-trace"
  | "pipeline-diff"
  | "persona-cards"
  | "dialectic"
  | "node-health"
  | "cost-estimator"
  | "testing-mode"
  | "edge-path-explorer"
  | "semantic-zoom"
  | "none";

export interface OverlayConfig {
  type: OverlayType;
  enabled: boolean;
  options?: Record<string, unknown>;
}

export interface NodeMetrics {
  nodeId: string;
  visitCount: number;
  avgDurationMs: number;
  errorCount: number;
  lastVisitedAt?: string;
  walkers: string[];
}

export interface EdgeMetrics {
  edgeId: string;
  traversalCount: number;
  avgEvalMs: number;
  trueCount: number;
  falseCount: number;
}

export interface CostEstimate {
  nodeId: string;
  tokenEstimate: number;
  costUSD: number;
  provider: string;
}

export interface WalkerTrace {
  walkerId: string;
  element: string;
  persona: string;
  path: Array<{
    nodeId: string;
    timestamp: string;
    durationMs: number;
  }>;
}

export interface PipelineDiff {
  added: { nodes: string[]; edges: string[] };
  removed: { nodes: string[]; edges: string[] };
  modified: { nodes: string[]; edges: string[] };
}
