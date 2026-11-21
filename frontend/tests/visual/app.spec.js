import { expect, test } from "@playwright/test";

test.use({ viewport: { width: 1280, height: 800 } });

// React 重写后 DOM 结构整体变化，快照基准随本分支重新生成
// （npm run test:visual:update）。fixture=states 的 DEV mock 沿用 backend.js。

test("empty repository view", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator(".window")).toHaveScreenshot("empty-light.png");
});

test("repository status cards", async ({ page }) => {
  await page.goto("/?fixture=states");
  await expect(page.locator(".window")).toHaveScreenshot(
    "repository-states.png",
  );
});

test("dark appearance", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/?fixture=states");
  await expect(page.locator(".window")).toHaveScreenshot(
    "repository-states-dark.png",
  );
});

test("logs and settings views", async ({ page }) => {
  await page.goto("/?fixture=states");
  await page.getByRole("button", { name: "运行日志" }).click();
  await expect(page.locator(".window")).toHaveScreenshot("logs.png");
  await page.getByRole("button", { name: "设置", exact: true }).click();
  await expect(page.locator(".window")).toHaveScreenshot("settings.png");
});

test("create mirror modal", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "新建镜像" }).click();
  await expect(page.locator("[data-slot=dialog-content]")).toHaveScreenshot(
    "create-modal.png",
  );
});

test("responsive boundary", async ({ page }) => {
  await page.setViewportSize({ width: 900, height: 800 });
  await page.goto("/?fixture=states");
  await expect(page.locator(".window")).toHaveScreenshot("responsive-900.png");
});
