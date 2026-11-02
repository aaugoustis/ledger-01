import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { expect, type Page, test } from "@playwright/test";
import * as XLSX from "xlsx";

const email = process.env.E2E_EMAIL ?? "e2e@example.com";
// A synthetic export from a made-up bank (shared with the Go tests).
const statement = fileURLToPath(
  new URL("../../../backend/internal/importer/testdata/northbank.csv", import.meta.url),
);

async function signIn(page: Page) {
  const res = await page.request.post("/auth/dev/login", { data: { email, name: "E2E Tester" } });
  expect(res.status()).toBe(204);
}

const unique = () => Math.random().toString(36).slice(2, 7).toUpperCase();

async function deleteRule(page: Page, pattern: string) {
  await page.goto("/settings/import-rules");
  await expect(page.getByRole("heading", { name: "Settings" })).toBeVisible();
  // Wait for the list (or its empty state) before looking for the rule.
  await expect(
    page.getByText("No rules yet").or(page.getByText(/^Contains “/).first()),
  ).toBeVisible();
  const remove = page.getByRole("button", { name: `Delete the rule for “${pattern}”` });
  if (await remove.count()) {
    await remove.click();
    await expect(remove).toHaveCount(0);
  }
}

test("import a bank CSV: columns, review, a rule, and duplicates next time", async ({
  page,
}, info) => {
  const walletName = `E2E Statement ${unique()}`;
  // Each project gets its own rule text (they run in parallel in one household).
  const pattern = info.project.name === "mobile" ? "Transit Card" : "City Transit";
  await signIn(page);
  await deleteRule(page, pattern);

  await page.goto("/wallets");
  await page.getByRole("button", { name: "Add wallet" }).first().click();
  await page.getByLabel("Name").fill(walletName);
  await page.getByRole("button", { name: "Create wallet" }).click();
  await expect(page.getByRole("heading", { name: walletName })).toBeVisible();

  // File → columns: the mapping is guessed from the headers and previewed.
  await page.getByRole("link", { name: "Import", exact: true }).click();
  await page.getByLabel("Statement file").setInputFiles(statement);
  await expect(page.getByLabel("Date format")).toContainText("yyyy-mm-dd");
  await expect(page.getByText("Acme Payroll", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Review rows" }).click();

  // Review: the repeated grocery row is a possible duplicate and starts unticked.
  await expect(page.getByText("6 rows · 5 ticked · 1 possible duplicate")).toBeVisible();
  // The Import button is pinned on screen, however long the list.
  await expect(page.getByRole("button", { name: /^Import \d+ transactions?$/ })).toBeInViewport();
  // The duplicate is folded into one line; showing settled rows lists it inline.
  await expect(page.getByLabel("Import row 3")).toHaveCount(0);
  await page.getByLabel("Show settled rows").check();
  await page.getByRole("button", { name: "Untick all" }).click();
  await expect(page.getByText("6 rows · 0 ticked · 1 possible duplicate")).toBeVisible();
  const row = (n: number) =>
    page.getByRole("listitem").filter({ has: page.getByLabel(`Import row ${n}`) });

  // Rent: pick a category.
  await page.getByLabel("Import row 6").check();
  await row(6).getByRole("combobox").click();
  await page.getByPlaceholder("Search categories…").fill("Home");
  await page.keyboard.press("Enter");
  await expect(row(6).getByRole("combobox")).toContainText("Home");

  // Transit: make a rule from the row; it fills the category in.
  await page.getByLabel("Import row 4").check();
  await page.getByRole("button", { name: "Make a rule from row 4" }).click();
  const dialog = page.getByRole("dialog", { name: "Make a rule" });
  await dialog.getByLabel("When the note contains").fill(pattern);
  await dialog.getByRole("combobox").click();
  await page.getByPlaceholder("Search categories…").fill("Transport");
  await page.keyboard.press("Enter");
  await dialog.getByRole("button", { name: "Save rule" }).click();
  await expect(row(4).getByRole("combobox")).toContainText("Transport");

  await page.getByRole("button", { name: "Import 2 transactions" }).click();
  await expect(page.getByText(`Imported 2 transactions into ${walletName}.`)).toBeVisible();

  // The wallet shows them (these September rows are in the past).
  await page.getByRole("link", { name: "Back to the wallet" }).click();
  await expect(
    page.getByText("−1,220.00 USD").or(page.getByText("-1,220.00 USD")).first(),
  ).toBeVisible();

  // The same file again: the saved columns skip straight to the review, and the imported rows
  // are now duplicates too.
  await page.getByRole("link", { name: "Import", exact: true }).click();
  await page.getByLabel("Statement file").setInputFiles(statement);
  await expect(page.getByText("Read with the columns saved for this wallet.")).toBeVisible();
  await expect(page.getByText("6 rows · 3 ticked · 3 possible duplicates")).toBeVisible();

  await deleteRule(page, pattern);
});

test("an old .xls statement is converted in the browser and imported", async ({ page }) => {
  const walletName = `E2E Legacy ${unique()}`;
  await signIn(page);
  await page.goto("/wallets");
  await page.getByRole("button", { name: "Add wallet" }).first().click();
  await page.getByLabel("Name").fill(walletName);
  await page.getByRole("button", { name: "Create wallet" }).click();
  await expect(page.getByRole("heading", { name: walletName })).toBeVisible();

  // A made-up statement saved as legacy .xls (BIFF8) with real date cells.
  const sheet = XLSX.utils.aoa_to_sheet([
    ["Date", "Description", "Amount"],
    [new Date(Date.UTC(2026, 8, 3)), "Corner Grocer", -45.2],
    [new Date(Date.UTC(2026, 8, 5)), "City Transit Card", -20],
    [new Date(Date.UTC(2026, 8, 30)), "Acme Payroll", 2400],
  ]);
  const book = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(book, sheet, "Statement");
  const xls = Buffer.from(XLSX.write(book, { bookType: "biff8", type: "array" }));

  await page.getByRole("link", { name: "Import", exact: true }).click();
  await page.getByLabel("Statement file").setInputFiles({
    name: "statement.xls",
    mimeType: "application/vnd.ms-excel",
    buffer: xls,
  });
  // The columns step shows the converted rows, dates included.
  await expect(page.getByText("Acme Payroll", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Review rows" }).click();
  await expect(page.getByText(/^3 rows · /)).toBeVisible();
  await expect(page.getByText("Corner Grocer", { exact: true })).toBeVisible();
});

test("a statement with bank details above the table starts at the header row", async ({ page }) => {
  const walletName = `E2E Decorated ${unique()}`;
  await signIn(page);
  await page.goto("/wallets");
  await page.getByRole("button", { name: "Add wallet" }).first().click();
  await page.getByLabel("Name").fill(walletName);
  await page.getByRole("button", { name: "Create wallet" }).click();
  await expect(page.getByRole("heading", { name: walletName })).toBeVisible();

  // A made-up export: bank name, holder and account lines, then the table.
  await page.getByRole("link", { name: "Import", exact: true }).click();
  await page
    .getByLabel("Statement file")
    .setInputFiles(
      fileURLToPath(
        new URL("../../../backend/internal/importer/testdata/decorated.csv", import.meta.url),
      ),
    );
  await expect(page.getByLabel("The table starts at")).toContainText("Row 6 · Fecha · Concepto");
  await expect(page.getByLabel("Date column")).toContainText("Fecha");
  await expect(page.getByText("Supermercado Central", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Review rows" }).click();
  await expect(page.getByText(/^3 rows · /)).toBeVisible();
});

test("a REDIVA discount merges into its purchase, and a renamed import is still recognised", async ({
  page,
}) => {
  const walletName = `E2E Pesos ${unique()}`;
  // A made-up statement: a purchase, its REDIVA 19210 VAT discount, and another purchase.
  const file = fileURLToPath(
    new URL("../../../backend/internal/importer/testdata/rediva.csv", import.meta.url),
  );
  await signIn(page);
  await page.goto("/wallets");
  await page.getByRole("button", { name: "Add wallet" }).first().click();
  await page.getByLabel("Name").fill(walletName);
  await page.getByRole("combobox").click();
  await page.getByPlaceholder("Search currency…").fill("UYU");
  await page.getByRole("option", { name: /^UYU/ }).click();
  await page.getByRole("button", { name: "Create wallet" }).click();
  await expect(page.getByRole("heading", { name: walletName })).toBeVisible();

  await page.getByRole("link", { name: "Import", exact: true }).click();
  await page.getByLabel("Statement file").setInputFiles(file);
  // The bank is detected from the descriptions.
  await expect(page.getByLabel("Bank")).toContainText("Itaú · Uruguay");
  await page.getByRole("button", { name: "Review rows" }).click();
  await expect(page.getByText("3 rows · 2 ticked · 1 discount merged")).toBeVisible();
  await expect(
    page.getByText("Includes a discount of 16.39: REDIVA 19210MERCADO SOL"),
  ).toBeVisible();
  // The discount row is folded into its purchase, not listed on its own.
  await expect(page.getByLabel("Import row 2")).toHaveCount(0);
  for (const n of [1, 3]) {
    const row = page.getByRole("listitem").filter({ has: page.getByLabel(`Import row ${n}`) });
    await row.getByRole("combobox").click();
    await page.getByPlaceholder("Search categories…").fill("Groceries");
    await page.keyboard.press("Enter");
    await expect(row.getByRole("combobox")).toContainText("Groceries");
    await expect(page.getByPlaceholder("Search categories…")).toHaveCount(0);
  }
  await page.getByRole("button", { name: "Import 2 transactions" }).click();
  await expect(page.getByText(`Imported 2 transactions into ${walletName}.`)).toBeVisible();

  // One expense for the purchase, net of its discount. Rename it.
  await page.getByRole("link", { name: "Back to the wallet" }).click();
  // The statement is from October 2026; the wallet opens on the current month.
  await expect(page.getByRole("heading", { name: walletName })).toBeVisible();
  await page.goto(`${new URL(page.url()).pathname}?from=2026-10-01&to=2026-10-31`);
  await page.getByText("COMPRA MERCADO SOL").click();
  const sheet = page.getByRole("dialog");
  await expect(sheet.getByLabel("Amount")).toHaveValue("483.61");
  await sheet.getByLabel("Note (optional)").fill("Weekly shop");
  await sheet.getByRole("button", { name: "Save changes" }).click();
  await expect(page.getByText("Weekly shop")).toBeVisible();

  // The same statement again: every row was imported before, the renamed one included.
  await page.getByRole("link", { name: "Import", exact: true }).click();
  await page.getByLabel("Statement file").setInputFiles(file);
  await expect(page.getByText("3 rows · 0 ticked · 3 possible duplicates")).toBeVisible();
  // They're folded into one line, which opens the Duplicates view.
  await page.getByRole("button", { name: /3 already imported/ }).click();
  // The purchase and its discount both point at the renamed transaction.
  await expect(page.getByText(/Imported before; now “Weekly shop”/)).toHaveCount(2);
});

test("a double charge and its refund start unticked; REDIVA 17934 merges too", async ({ page }) => {
  const walletName = `E2E Refund ${unique()}`;
  await signIn(page);
  await page.goto("/wallets");
  await page.getByRole("button", { name: "Add wallet" }).first().click();
  await page.getByLabel("Name").fill(walletName);
  await page.getByRole("button", { name: "Create wallet" }).click();
  await expect(page.getByRole("heading", { name: walletName })).toBeVisible();

  // A made-up statement: charged twice and refunded once, then a meal with its VAT discount.
  await page.getByRole("link", { name: "Import", exact: true }).click();
  await page
    .getByLabel("Statement file")
    .setInputFiles(
      fileURLToPath(
        new URL("../../../backend/internal/importer/testdata/refund.csv", import.meta.url),
      ),
    );
  await expect(page.getByLabel("Bank")).toContainText("Itaú · Uruguay");
  await page.getByRole("button", { name: "Review rows" }).click();
  await expect(
    page.getByText("5 rows · 2 ticked · 1 refunded purchase left out · 1 discount merged"),
  ).toBeVisible();
  await expect(
    page.getByText("Includes a discount of 99.00: REDIVA 17934PARRILLA CENTRO"),
  ).toBeVisible();
  // The refund pair is folded into one line; it opens the Left out view.
  await page.getByRole("button", { name: /2 refund rows left out/ }).click();
  await expect(page.getByText("Refunded in full by row 3")).toBeVisible();
  await expect(page.getByText("Refund of row 2")).toBeVisible();
  await expect(page.getByLabel("Import row 2")).not.toBeChecked();
  await expect(page.getByLabel("Import row 3")).not.toBeChecked();

  // Without the bank's wording nothing is recognised: the refund and discount are plain rows,
  // and the second charge is only a repeat.
  await page.getByRole("button", { name: "Change columns" }).first().click();
  await page.getByLabel("Bank").click();
  await page.getByRole("option", { name: "Other bank" }).click();
  await page.getByRole("button", { name: "Review rows" }).click();
  await expect(page.getByText("5 rows · 4 ticked · 1 possible duplicate")).toBeVisible();
});

test("a statement can be dropped anywhere on the page", async ({ page }) => {
  const walletName = `E2E Drop ${unique()}`;
  await signIn(page);
  await page.goto("/wallets");
  await page.getByRole("button", { name: "Add wallet" }).first().click();
  await page.getByLabel("Name").fill(walletName);
  await page.getByRole("button", { name: "Create wallet" }).click();
  await expect(page.getByRole("heading", { name: walletName })).toBeVisible();
  await page.getByRole("link", { name: "Import", exact: true }).click();
  await expect(page.getByLabel("Statement file")).toBeEnabled();

  // A made-up statement, dragged over the page's heading and dropped there.
  const csv = readFileSync(statement, "utf8");
  const files = await page.evaluateHandle((text) => {
    const dt = new DataTransfer();
    dt.items.add(new File([text], "northbank.csv", { type: "text/csv" }));
    return dt;
  }, csv);
  const heading = page.getByRole("heading", { name: "Import" });
  await heading.dispatchEvent("dragenter", { dataTransfer: files });
  await expect(page.getByText("Drop the statement")).toBeVisible();
  await heading.dispatchEvent("drop", { dataTransfer: files });
  await expect(page.getByText("Drop the statement")).toHaveCount(0);
  await expect(page.getByText("Acme Payroll", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Review rows" })).toBeVisible();
});
