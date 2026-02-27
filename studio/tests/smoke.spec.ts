import { test, expect } from "@playwright/test";
import { getBridgeSnapshot, waitForGraph, bridgeCall } from "./helpers";

test.describe("Studio Smoke Tests", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
  });

  test("graph renders with demo pipeline", async ({ page }) => {
    await waitForGraph(page);
    const snap = await getBridgeSnapshot(page);
    expect(snap.nodeCount).toBeGreaterThanOrEqual(4);
    expect(snap.edgeCount).toBeGreaterThanOrEqual(3);
  });

  test("bridge is available on window", async ({ page }) => {
    await page.waitForFunction(() => window.__origami !== undefined);
    const count = await bridgeCall(page, "nodeCount");
    expect(typeof count).toBe("number");
  });

  test("node can be selected", async ({ page }) => {
    await waitForGraph(page);
    const node = page.locator(".react-flow__node").first();
    await node.click();
    const selected = await bridgeCall(page, "selectedNode");
    expect(selected).not.toBeNull();
  });

  test("graph has correct structure", async ({ page }) => {
    await waitForGraph(page);
    const snap = await getBridgeSnapshot(page);
    expect(snap.pipeline).toBe("demo");
  });

  test("minimap is visible", async ({ page }) => {
    await waitForGraph(page);
    const minimap = page.locator(".react-flow__minimap");
    await expect(minimap).toBeVisible();
  });

  test("controls are visible", async ({ page }) => {
    await waitForGraph(page);
    const controls = page.locator(".react-flow__controls");
    await expect(controls).toBeVisible();
  });
});
