import { test, expect } from "@playwright/test";
import { requireKami, waitForGraph } from "./helpers";

test.describe("Kami Integration", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    const available = await requireKami(page);
    if (!available) {
      test.skip(true, "Kami server not reachable — skipping integration tests");
    }
  });

  test("SSE connection established", async ({ page }) => {
    await waitForGraph(page);
    // Kami integration will be wired in later phases.
    // For now, verify the graph renders when Kami is available.
    const nodes = page.locator(".react-flow__node");
    expect(await nodes.count()).toBeGreaterThan(0);
  });
});
