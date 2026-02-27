import { type Page, expect } from "@playwright/test";
import type { OrigamiBridge, PipelineSnapshot } from "../src/bridge";

/**
 * Wait for the __origami bridge to be available and return a snapshot.
 */
export async function getBridgeSnapshot(
  page: Page
): Promise<PipelineSnapshot> {
  return page.evaluate(() => window.__origami.snapshot());
}

/**
 * Get a single bridge value.
 */
export async function bridgeCall<K extends keyof OrigamiBridge>(
  page: Page,
  method: K
): Promise<ReturnType<OrigamiBridge[K]>> {
  return page.evaluate(
    (m) => (window.__origami[m] as () => unknown)(),
    method
  ) as Promise<ReturnType<OrigamiBridge[K]>>;
}

/**
 * Check if Kami is reachable. If not, skip the test gracefully.
 */
export async function requireKami(
  page: Page,
  port = 9800
): Promise<boolean> {
  try {
    const resp = await page.request.get(`http://localhost:${port}/metrics`);
    return resp.ok();
  } catch {
    return false;
  }
}

/**
 * Wait for the graph to render at least one node.
 */
export async function waitForGraph(page: Page) {
  await page.waitForSelector(".react-flow__node", { timeout: 10_000 });
  const snap = await getBridgeSnapshot(page);
  expect(snap.nodeCount).toBeGreaterThan(0);
}
