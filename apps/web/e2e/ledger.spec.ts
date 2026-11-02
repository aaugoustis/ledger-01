import { expect, type Page, test } from "@playwright/test";

const email = process.env.E2E_EMAIL ?? "e2e@example.com";

// Signs in through the dev-only endpoint (proxied by Next to the API) and returns with the
// session cookie in the browser context.
async function signIn(page: Page) {
  const res = await page.request.post("/auth/dev/login", { data: { email, name: "E2E Tester" } });
  expect(res.status()).toBe(204);
}

const unique = () => Math.random().toString(36).slice(2, 7).toUpperCase();

test("signed-out visitors are sent to the login page", async ({ page }) => {
  await page.goto("/wallets");
  await expect(page).toHaveURL(/\/login\?next=%2Fwallets/);
  await expect(page.getByRole("link", { name: "Continue with Google" })).toBeVisible();
});

test("wallets, an expense with a label, and a USD→UYU transfer", async ({ page }, info) => {
  const tag = unique();
  const usdName = `E2E USD ${tag}`;
  const uyuName = `E2E UYU ${tag}`;
  await signIn(page);

  // Create the UYU wallet first, then USD (we stay on the USD wallet afterwards).
  for (const [name, code] of [
    [uyuName, "UYU"],
    [usdName, "USD"],
  ] as const) {
    await page.goto("/wallets");
    await page.getByRole("button", { name: "Add wallet" }).first().click();
    await page.getByLabel("Name").fill(name);
    if (code !== "USD") {
      await page.getByRole("combobox").click();
      await page.getByPlaceholder("Search currency…").fill(code);
      await page.getByRole("option", { name: new RegExp(`^${code}`) }).click();
    }
    await page.getByLabel("Initial balance (optional)").fill(code === "USD" ? "500" : "0");
    await page.getByRole("button", { name: "Create wallet" }).click();
    await expect(page.getByRole("heading", { name })).toBeVisible();
  }

  // Expense with a new label.
  const open =
    info.project.name === "mobile"
      ? page.getByRole("button", { name: "New transaction" }).last()
      : page.getByRole("button", { name: "New transaction" }).first();
  await open.click();
  const sheet = page.getByRole("dialog");
  const categoryTrigger = sheet.getByRole("combobox").first();

  // Form flow: the sheet opens on its first field, and since the category is empty its list is
  // already open with the search focused. Escape closes it back onto the trigger.
  await expect(page.getByPlaceholder("Search categories…")).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(categoryTrigger).toBeFocused();

  // Submitting an empty expense faults the category picker: it animates (data-fault), takes
  // focus and its label turns red with a fault LED; editing it clears the error.
  await sheet.getByRole("button", { name: "Add transaction" }).click();
  await expect(categoryTrigger).toHaveAttribute("data-fault", "");
  await expect(categoryTrigger).toBeFocused();
  await expect(sheet.getByRole("alert").filter({ hasText: "Choose a category" })).toHaveCount(1);
  await expect(sheet.locator("label[data-invalid]", { hasText: "Category" })).toBeVisible();

  // Type to search: a letter on the focused trigger opens it with that search; Enter picks and
  // focus moves on to the amount.
  await page.keyboard.press("g");
  const search = page.getByPlaceholder("Search categories…");
  await expect(search).toHaveValue("g");
  await search.pressSequentially("roc");
  await page.keyboard.press("Enter");
  await expect(categoryTrigger).toContainText("Groceries");
  await expect(sheet.getByLabel("Amount")).toBeFocused();
  await expect(sheet.getByRole("alert").filter({ hasText: "Choose a category" })).toHaveCount(0);

  // Still no amount: the amount well faults next.
  await sheet.getByRole("button", { name: "Add transaction" }).click();
  await expect(sheet.locator("[data-slot=money-input]")).toHaveAttribute("data-fault", "");
  await expect(sheet.getByRole("alert").filter({ hasText: "Enter an amount" })).toHaveCount(1);
  await expect(sheet.locator("label[data-invalid]", { hasText: "Amount" })).toBeVisible();
  await sheet.getByLabel("Amount").fill("12.34");
  // Tab follows the form's order: amount → date.
  await page.keyboard.press("Tab");
  await expect(sheet.getByLabel("Date")).toBeFocused();
  await sheet.getByLabel("Note (optional)").fill(`Market ${tag}`);
  await sheet.getByLabel("Labels (optional)").click();
  await page.getByPlaceholder("Search or create…").fill(`Trip ${tag}`);
  await page.getByRole("option", { name: new RegExp(`Add label “Trip ${tag}”`) }).click();
  await page.keyboard.press("Escape");
  await sheet.getByRole("button", { name: "Add transaction" }).click();
  await expect(page.getByText(`Market ${tag}`)).toBeVisible();

  // Cross-currency transfer with the three-field converter: sent + received derive the rate.
  await open.click();
  await sheet.getByRole("tab", { name: "Transfer" }).click();
  await sheet.getByLabel("To wallet").click();
  await page.getByRole("option", { name: new RegExp(uyuName) }).click();
  await sheet.getByLabel("Sent (USD)").fill("100.00");
  await sheet.getByLabel("Received (UYU)").fill("3905.00");
  await expect(sheet.getByLabel("Rate (UYU per USD)")).toHaveValue("39.050000");
  await sheet.getByRole("button", { name: "Add transfer" }).click();

  // USD: 500 − 12.34 − 100 = 387.66
  await expect(page.getByText(new RegExp(`To\\s*${uyuName}`))).toBeVisible();
  await expect(page.getByText("+387.66 USD").first()).toBeVisible();

  // UYU wallet received 3,905.00.
  await page.goto("/wallets");
  await page.getByRole("link", { name: new RegExp(uyuName) }).click();
  await expect(page.getByText("+3,905.00 UYU").first()).toBeVisible();
  await expect(page.getByText(new RegExp(`From\\s*${usdName}`))).toBeVisible();

  // Dashboard: household totals in the base currency, charts, and the category drill-down.
  // On phones the brand mark is the way home.
  await (info.project.name === "mobile"
    ? page.getByRole("link", { name: "L/01" })
    : page.getByRole("link", { name: "Dashboard" })
  ).click();
  await expect(page.getByRole("heading", { name: "Dashboard" })).toBeVisible();
  await expect(page.getByText("Total balance")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Expenses by category" })).toBeVisible();
  const groceries = page.getByRole("link", { name: /Groceries/ }).first();
  await expect(groceries).toBeVisible();
  await groceries.click();
  await expect(page.getByRole("heading", { name: "Groceries" })).toBeVisible();
  await expect(page.getByText("Subtotal")).toBeVisible();
  await expect(page.getByText(`Market ${tag}`)).toBeVisible();

  // Budget: a monthly limit on groceries from the USD wallet counts the 12.34 expense.
  const budgetName = `Food ${tag}`;
  await page.getByRole("link", { name: "Budgets" }).click();
  await page.getByRole("button", { name: "Add budget" }).first().click();
  const editor = page.getByRole("dialog");
  // The sheet opens on its first field.
  await expect(editor.getByLabel("Name")).toBeFocused();
  // Submitting without an amount faults the amount field.
  await editor.getByLabel("Name").fill(budgetName);
  await editor.getByRole("button", { name: "Add budget" }).click();
  await expect(editor.locator("label[data-invalid]", { hasText: "Amount" })).toBeVisible();
  await editor.getByLabel("Amount").fill("100");
  await editor.getByLabel("Wallets").click();
  await page.getByPlaceholder("Search wallets…").fill(usdName);
  await page.getByRole("option", { name: new RegExp(usdName) }).click();
  await page.keyboard.press("Escape");
  await expect(editor.getByLabel("Wallets")).toContainText(usdName);
  await editor.getByLabel("Categories").click();
  await page.getByPlaceholder("Search categories…").fill("Groceries");
  await page.getByRole("option", { name: /Groceries/ }).click();
  await page.keyboard.press("Escape");
  await editor.getByRole("button", { name: "Add budget" }).click();

  // The detail page shows the period's spending and the expense.
  await expect(page.getByRole("heading", { name: budgetName })).toBeVisible();
  await expect(page.getByText("12.34 USD of 100.00 USD spent")).toBeVisible();
  await expect(page.getByText("87.66 USD left")).toBeVisible();
  await expect(page.getByText(`Market ${tag}`)).toBeVisible();
  await expect(page.getByRole("meter", { name: `${budgetName} budget` })).toHaveAttribute(
    "aria-valuenow",
    "12",
  );

  // Editing the amount below what's spent shows it over budget.
  await page.getByRole("button", { name: "Edit" }).click();
  await editor.getByLabel("Amount").fill("10");
  await editor.getByRole("button", { name: "Save budget" }).click();
  await expect(page.getByText("Over by 2.34 USD").first()).toBeVisible();

  await page.getByRole("link", { name: "Budgets" }).first().click();
  await expect(page.getByRole("link", { name: new RegExp(budgetName) })).toContainText(
    "Over by 2.34 USD",
  );
});
