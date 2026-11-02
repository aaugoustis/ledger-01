import { expect, type Locator, type Page, test } from "@playwright/test";

const email = process.env.E2E_EMAIL ?? "e2e@example.com";

async function signIn(page: Page) {
  const res = await page.request.post("/auth/dev/login", { data: { email, name: "E2E Tester" } });
  expect(res.status()).toBe(204);
}

test("the app is installable", async ({ page, context }) => {
  await signIn(page);
  await page.goto("/dashboard");
  await expect(page.getByRole("heading", { name: "Dashboard" })).toBeVisible();
  // Chrome's own installability check (what decides whether it offers to install the app).
  const cdp = await context.newCDPSession(page);
  const manifest = await cdp.send("Page.getAppManifest");
  expect(manifest.url).toMatch(/\/manifest\.webmanifest$/);
  expect(manifest.errors).toEqual([]);
  const { installabilityErrors } = await cdp.send("Page.getInstallabilityErrors");
  // Test browsers run in throwaway (incognito-like) profiles, which Chrome never installs into;
  // that one says nothing about the app.
  expect(installabilityErrors.filter((e) => e.errorId !== "in-incognito")).toEqual([]);
});

test.describe("on a 375×812 phone", () => {
  test.use({ viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true });

  test("an expense takes at most four taps once the wallet is open", async ({ page }, info) => {
    test.skip(info.project.name !== "mobile", "a phone flow");
    await signIn(page);
    const res = await page.request.post("/api/ledger.v1.WalletService/CreateWallet", {
      data: {
        name: `E2E Taps ${Date.now() % 100000}`,
        currency: "USD",
        initialBalanceMinor: "10000",
      },
    });
    expect(res.ok()).toBe(true);
    const { wallet } = await res.json();
    await page.goto(`/wallets/${wallet.id}`);
    await expect(page.getByRole("heading", { name: wallet.name })).toBeVisible();

    let taps = 0;
    const tap = async (target: Locator) => {
      await target.tap();
      taps++;
    };
    const sheet = page.getByRole("dialog", { name: "New transaction" });

    // 1: the quick-add button. The sheet opens with the category list open and searchable.
    await tap(page.getByRole("button", { name: "New transaction" }).last());
    await expect(page.getByPlaceholder("Search categories…")).toBeFocused();
    // 2: a category. Focus moves on to the amount, so typing needs no tap.
    await tap(page.getByRole("option", { name: "Groceries" }));
    await expect(sheet.getByLabel("Amount")).toBeFocused();
    await page.keyboard.type("12.50");
    // 3: add. The date defaults to today.
    await tap(sheet.getByRole("button", { name: "Add transaction" }));

    await expect(sheet).toBeHidden();
    await expect(page.getByText("+87.50 USD").first()).toBeVisible();
    expect(taps).toBeLessThanOrEqual(4);
  });
});
