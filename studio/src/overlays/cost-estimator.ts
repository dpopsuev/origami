import type { CostEstimate } from "./types";

export interface CostSummary {
  totalTokens: number;
  totalCostUSD: number;
  perNode: CostEstimate[];
  warningThresholdUSD: number;
  exceedsThreshold: boolean;
}

/**
 * Aggregate cost estimates across nodes and check against threshold.
 */
export function computeCostSummary(
  estimates: CostEstimate[],
  warningThresholdUSD = 1.0
): CostSummary {
  let totalTokens = 0;
  let totalCostUSD = 0;

  for (const e of estimates) {
    totalTokens += e.tokenEstimate;
    totalCostUSD += e.costUSD;
  }

  return {
    totalTokens,
    totalCostUSD,
    perNode: estimates,
    warningThresholdUSD,
    exceedsThreshold: totalCostUSD > warningThresholdUSD,
  };
}

/**
 * Format cost as human-readable string.
 */
export function formatCost(usd: number): string {
  if (usd < 0.01) return `$${(usd * 100).toFixed(2)}¢`;
  return `$${usd.toFixed(4)}`;
}

/**
 * Format token count with K/M suffix.
 */
export function formatTokens(count: number): string {
  if (count >= 1_000_000) return `${(count / 1_000_000).toFixed(1)}M`;
  if (count >= 1_000) return `${(count / 1_000).toFixed(1)}K`;
  return `${count}`;
}
