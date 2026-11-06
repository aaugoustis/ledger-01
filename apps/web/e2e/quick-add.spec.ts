import { expect, type Locator, type Page, test } from "@playwright/test";

const email = process.env.E2E_EMAIL ?? "e2e@example.com";

async function signIn(page: Page) {
  const res = await page.request.post("/auth/dev/login", { data: { email, name: "E2E Tester" } });
  expect(res.status()).toBe(204);
}

async function createWallet(page: Page, name: string) {
  const res = await page.request.post("/api/ledger.v1.WalletService/CreateWallet", {
    data: { name, currency: "USD", initialBalanceMinor: "10000" },
  });
  expect(res.ok()).toBe(true);
  return (await res.json()).wallet as { id: string; name: string };
}

const unique = () => Math.random().toString(36).slice(2, 7).toUpperCase();

test("the app icon's shortcut opens a new transaction on the dashboard", async ({ page }, info) => {
  await signIn(page);
  await createWallet(page, `E2E Shortcut ${unique()}`);

  const manifest = await (await page.request.get("/manifest.webmanifest")).json();
  const shortcut = manifest.shortcuts?.[0];
  expect(shortcut).toMatchObject({ name: "New transaction", url: "/dashboard?new=transaction" });

  await page.goto(shortcut.url);
  const sheet = page.getByRole("dialog", { name: "New transaction" });
  await expect(sheet).toBeVisible();
  await expect(page.getByPlaceholder("Search categories…")).toBeFocused();
  // The parameter is dropped, so a reload shows the plain dashboard.
  await expect(page).toHaveURL(/\/dashboard$/);
  await page.reload();
  await expect(page.getByRole("heading", { name: "Dashboard" })).toBeVisible();
  await expect(sheet).toBeHidden();

  // The dashboard's own button opens it too: a button on desktop, the pinned one on phones.
  const button = page.getByRole("button", { name: "New transaction" });
  await (info.project.name === "mobile" ? button.last() : button.first()).click();
  await expect(sheet).toBeVisible();
});

test.describe("on a 375×812 phone", () => {
  test.use({ viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true });

  test("from the dashboard, an expense takes three taps in the wallet last used", async ({
    page,
  }, info) => {
    test.skip(info.project.name !== "mobile", "a phone flow");
    const tag = unique();
    await signIn(page);
    await createWallet(page, `E2E Card ${tag}`);
    const cash = await createWallet(page, `E2E Cash ${tag}`);
    const sheet = page.getByRole("dialog", { name: "New transaction" });
    const quickAdd = page.getByRole("button", { name: "New transaction" }).last();

    // The first one goes to cash, picked in the Wallet field below the amount.
    await page.goto("/dashboard");
    await quickAdd.tap();
    await page.getByRole("option", { name: "Groceries" }).tap();
    await expect(sheet.getByLabel("Amount")).toBeFocused();
    await page.keyboard.type("7.25");
    await sheet.getByLabel("Wallet").tap();
    await page.getByRole("option", { name: new RegExp(cash.name) }).tap();
    await sheet.getByRole("button", { name: "Add transaction" }).tap();
    await expect(sheet).toBeHidden();

    // The next one starts in cash: the quick-add button, a category and add.
    let taps = 0;
    const tap = async (target: Locator) => {
      await target.tap();
      taps++;
    };
    await tap(quickAdd);
    await expect(sheet.getByLabel("Wallet")).toContainText(cash.name);
    await tap(page.getByRole("option", { name: "Groceries" }));
    await expect(sheet.getByLabel("Amount")).toBeFocused();
    await page.keyboard.type("4.50");
    await tap(sheet.getByRole("button", { name: "Add transaction" }));
    await expect(sheet).toBeHidden();
    expect(taps).toBeLessThanOrEqual(3);

    // Cash: 100 − 7.25 − 4.50 = 88.25
    await page.goto(`/wallets/${cash.id}`);
    await expect(page.getByText("+88.25 USD").first()).toBeVisible();
  });
});
