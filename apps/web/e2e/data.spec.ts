import { readFile } from "node:fs/promises";
import { expect, type Page, test } from "@playwright/test";

// A household of its own per project: restoring replaces everything in it.
const emailFor = (project: string) => `e2e-backup-${project}@example.com`;

async function signIn(page: Page, email: string) {
  const res = await page.request.post("/auth/dev/login", { data: { email, name: "E2E Backup" } });
  expect(res.status()).toBe(204);
}

async function createWallet(page: Page, name: string) {
  const res = await page.request.post("/api/ledger.v1.WalletService/CreateWallet", {
    data: { name, currency: "USD", initialBalanceMinor: "12345" },
  });
  expect(res.ok()).toBe(true);
}

test("download a backup, restore it over newer data, export CSV", async ({ page }, info) => {
  await signIn(page, emailFor(info.project.name));
  const tag = Date.now() % 100000;
  const kept = `E2E Kept ${tag}`;
  const later = `E2E Later ${tag}`;
  await createWallet(page, kept);

  await page.goto("/settings/data");
  await expect(page.getByRole("link", { name: /Data/ })).toHaveAttribute("aria-current", "page");
  const [backup] = await Promise.all([
    page.waitForEvent("download"),
    page.getByRole("button", { name: "Download backup" }).click(),
  ]);
  expect(backup.suggestedFilename()).toMatch(/^ledger-01-backup-\d{4}-\d{2}-\d{2}\.json\.gz$/);
  const file = await backup.path();

  // A wallet added after the backup is gone once it's restored.
  await createWallet(page, later);
  await page.getByLabel("Backup file").setInputFiles({
    name: backup.suggestedFilename(),
    mimeType: "application/gzip",
    buffer: await readFile(file),
  });
  // The file's contents are previewed before anything is replaced.
  await expect(page.getByRole("term").filter({ hasText: /^Transactions$/ })).toBeVisible();
  await page.getByRole("button", { name: "Restore backup" }).click();
  const dialog = page.getByRole("alertdialog");
  await expect(dialog.getByText("Replace all data in this household?")).toBeVisible();
  await dialog.getByRole("button", { name: "Replace data" }).click();
  await expect(page.getByText(/^Restored \d+ transactions$/)).toBeVisible();

  await page.goto("/wallets");
  await expect(page.getByText(kept).first()).toBeVisible();
  await expect(page.getByText(later)).toHaveCount(0);

  // CSV export downloads only when there's something in it.
  await page.goto("/settings/data");
  const tx = await page.request.post("/api/ledger.v1.WalletService/ListWallets", { data: {} });
  const wallet = (await tx.json()).wallets.find((w: { name: string }) => w.name === kept);
  const cats = await page.request.post("/api/ledger.v1.CategoryService/ListCategories", {
    data: {},
  });
  const category = (await cats.json()).categories.find(
    (c: { kind: string }) => c.kind === "CATEGORY_KIND_EXPENSE",
  );
  const created = await page.request.post("/api/ledger.v1.TransactionService/CreateTransaction", {
    data: {
      walletId: wallet.id,
      kind: "TRANSACTION_KIND_EXPENSE",
      categoryId: category.id,
      amountMinor: "1250",
      occurredOn: "2026-10-01",
      note: "E2E coffee",
    },
  });
  expect(created.ok()).toBe(true);
  const [csv] = await Promise.all([
    page.waitForEvent("download"),
    page.getByRole("button", { name: "Export transactions" }).click(),
  ]);
  expect(csv.suggestedFilename()).toMatch(/^ledger-01-transactions-.*\.csv$/);
  const text = await readFile(await csv.path(), "utf8");
  expect(text).toContain("Date,Wallet,Type,Category,Amount,Currency,Labels,Note,Transfer wallet");
  expect(text).toContain(`${kept},expense,`);
  expect(text).toContain("-12.50,USD,,E2E coffee,");
});
