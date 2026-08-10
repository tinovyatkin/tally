import { copyFileSync } from "node:fs";
import { join } from "node:path";

import { expect, test } from "./fixtures";
import { openFile, openWorkspace, problemRows, readEditorText, runCommand } from "./utils";

const capture = process.env.UPDATE_MARKETPLACE_ASSETS === "true";
const assetsDir = join(__dirname, "..", "assets");
const marketplaceDockerfile = join(__dirname, "fixtures", "Dockerfile.marketplace");
const marketplaceApp = join(__dirname, "fixtures", "app");

test("capture Marketplace screenshots from the packaged extension", async ({
  sharedCodeServer,
  projectDir,
  page,
}) => {
  test.skip(!capture, "set UPDATE_MARKETPLACE_ASSETS=true to refresh Marketplace screenshots");
  test.setTimeout(120_000);

  copyFileSync(marketplaceDockerfile, join(projectDir, "Dockerfile"));
  copyFileSync(marketplaceApp, join(projectDir, "app"));
  await openWorkspace(page, sharedCodeServer.url, projectDir);
  await openFile(page, "Dockerfile");
  const secondarySidebarButton = page.getByRole("button", {
    name: /Toggle Secondary Side Bar/,
  });
  if (
    (await secondarySidebarButton.count()) > 0 &&
    (await secondarySidebarButton.first().getAttribute("aria-pressed")) === "true"
  ) {
    await secondarySidebarButton.first().click();
  }
  await runCommand(page, "Problems: Focus on Problems View");

  const rows = problemRows(page);
  await expect.poll(() => rows.count(), { timeout: 30_000 }).toBeGreaterThan(0);
  await expect(rows.first()).toBeVisible();
  const before = await rows.count();

  await page.screenshot({
    path: join(assetsDir, "marketplace-diagnostics.png"),
    animations: "disabled",
  });

  const copyLine = page.locator(".monaco-editor .view-line").filter({ hasText: "copy app /app" });
  await copyLine.first().click();
  await runCommand(page, "Quick Fix...");

  const actions = page.locator(".action-widget .monaco-list-row.action");
  await expect(actions.first()).toBeVisible({ timeout: 15_000 });
  await page.screenshot({
    path: join(assetsDir, "marketplace-quick-fix.png"),
    animations: "disabled",
  });

  await page.keyboard.press("Escape");
  await runCommand(page, "Tally: Fix all auto-fixable issues");
  await expect
    .poll(async () => await readEditorText(page), { timeout: 20_000 })
    .toContain("COPY app /app");
  await expect
    .poll(async () => await readEditorText(page), { timeout: 20_000 })
    .toContain("WORKDIR /");
  await expect.poll(() => rows.count(), { timeout: 20_000 }).toBeLessThan(before);
  await page.getByRole("code").first().click();
  await page.keyboard.press("ArrowRight");
  await page
    .getByRole("button", { name: /Toggle Panel/ })
    .first()
    .click();

  await page.screenshot({
    path: join(assetsDir, "marketplace-fixed.png"),
    animations: "disabled",
  });
});
