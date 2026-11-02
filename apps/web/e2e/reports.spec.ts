import { expect, type Page, test } from "@playwright/test";

const email = process.env.E2E_EMAIL ?? "e2e@example.com";

async function signIn(page: Page) {
  const res = await page.request.post("/auth/dev/login", { data: { email, name: "E2E Tester" } });
  expect(res.status()).toBe(204);
}

async function rpc<T>(page: Page, method: string, data: unknown): Promise<T> {
  const res = await page.request.post(`/api/ledger.v1.${method}`, { data });
  expect(res.ok(), await res.text()).toBe(true);
  return res.json();
}

const localToday = () => {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
};

test("a wallet's overview, narrowed by note and amount", async ({ page }) => {
  await signIn(page);
  const { wallet } = await rpc<{ wallet: { id: string; name: string } }>(
    page,
    "WalletService/CreateWallet",
    { name: `E2E Overview ${Date.now() % 100000}`, currency: "USD", initialBalanceMinor: "100000" },
  );
  const { categories } = await rpc<{ categories: { id: string; name: string }[] }>(
    page,
    "CategoryService/ListCategories",
    {},
  );
  const groceries = categories.find((c) => c.name === "Groceries");
  for (const [amount, note] of [
    ["1200", "Corner Grocer"],
    ["8000", "Corner grocer weekly shop"],
    ["500", "Coffee"],
  ] as const) {
    await rpc(page, "TransactionService/CreateTransaction", {
      walletId: wallet.id,
      categoryId: groceries?.id,
      amountMinor: amount,
      occurredOn: localToday(),
      note,
      today: localToday(),
    });
  }

  // The Overview tab: the dashboard for this wallet alone, in its currency.
  await page.goto(`/wallets/${wallet.id}`);
  await page.getByRole("link", { name: "Overview" }).click();
  await expect(page.getByRole("link", { name: "Overview" })).toHaveAttribute(
    "aria-current",
    "page",
  );
  const expenses = page.locator("dl > div").filter({ hasText: "Period expenses" });
  await expect(expenses).toContainText(/97\.00 USD/);
  await expect(page.getByRole("button", { name: /wallets/i })).toHaveCount(0);

  // Note search (case-insensitive), then an amount floor on top.
  await page.getByLabel("Search notes").fill("GROCER");
  await expect(expenses).toContainText(/92\.00 USD/);
  await expect(page).toHaveURL(/note=GROCER/);
  await page.getByRole("button", { name: "Amount" }).click();
  await page.getByLabel("Min amount").fill("50");
  await page.keyboard.press("Escape");
  await expect(expenses).toContainText(/80\.00 USD/);
  // Balances ignore the filters.
  await expect(page.locator("dl > div").filter({ hasText: "Total balance" })).toContainText(
    /903\.00 USD/,
  );

  // The category drill-down keeps the wallet, its currency and the filters.
  await page
    .getByRole("link", { name: /Groceries/ })
    .first()
    .click();
  await expect(page.getByRole("heading", { name: "Groceries" })).toBeVisible();
  await expect(page.getByText("Corner grocer weekly shop")).toBeVisible();
  await expect(page.getByText("Coffee")).toHaveCount(0);

  // The household dashboard has the same filters.
  await page.goto("/dashboard?note=no-such-note-anywhere");
  await expect(page.getByLabel("Search notes")).toHaveValue("no-such-note-anywhere");
  await expect(page.getByText("No expenses in this period.")).toBeVisible();
});
