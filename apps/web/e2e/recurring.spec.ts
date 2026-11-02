import { expect, type Page, test } from "@playwright/test";

const email = process.env.E2E_EMAIL ?? "e2e@example.com";

async function signIn(page: Page) {
  const res = await page.request.post("/auth/dev/login", { data: { email, name: "E2E Tester" } });
  expect(res.status()).toBe(204);
}

const unique = () => Math.random().toString(36).slice(2, 7).toUpperCase();

test("a monthly expense: upcoming in Future, edited this and future, never in the balance", async ({
  page,
}, info) => {
  const tag = unique();
  const walletName = `E2E Recurring ${tag}`;
  const note = `Rent ${tag}`;
  await signIn(page);

  await page.goto("/wallets");
  await page.getByRole("button", { name: "Add wallet" }).first().click();
  await page.getByLabel("Name").fill(walletName);
  await page.getByLabel("Initial balance (optional)").fill("1000");
  await page.getByRole("button", { name: "Create wallet" }).click();
  await expect(page.getByRole("heading", { name: walletName })).toBeVisible();

  // A monthly expense from today. The category list opens on its own; type to pick.
  const open =
    info.project.name === "mobile"
      ? page.getByRole("button", { name: "New transaction" }).last()
      : page.getByRole("button", { name: "New transaction" }).first();
  await open.click();
  const sheet = page.getByRole("dialog");
  await page.getByPlaceholder("Search categories…").fill("Home");
  await page.keyboard.press("Enter");
  // Picking moves focus on to the amount; wait for it before typing elsewhere.
  await expect(sheet.getByLabel("Amount")).toBeFocused();
  await sheet.getByLabel("Amount").fill("100");
  await sheet.getByLabel("Note (optional)").fill(note);
  await sheet.getByLabel("Repeat").click();
  await page.getByRole("option", { name: "Every month" }).click();
  await expect(sheet.getByLabel("Ends")).toBeVisible();
  await sheet.getByRole("button", { name: "Add transaction" }).click();

  // Today's occurrence counts; the upcoming ones don't. The row carries the repeat marker.
  await expect(page.getByText("+900.00 USD").first()).toBeVisible();
  const row = page.getByRole("button", { name: new RegExp(note) });
  await expect(row).toHaveCount(1);
  await expect(row.getByText("Repeats: Every month")).toBeAttached();

  // Future: the next three months, soonest first.
  await page.getByRole("link", { name: "Future" }).click();
  await expect(page.getByRole("link", { name: "Future" })).toHaveAttribute("aria-current", "page");
  const upcoming = page.getByRole("button", { name: new RegExp(note) });
  await expect(upcoming).toHaveCount(3);

  // Edit the first upcoming one: this and future ones.
  await upcoming.first().click();
  await expect(sheet.getByLabel("Repeat")).toContainText("Every month");
  await sheet.getByLabel("Amount").fill("120");
  await sheet.getByRole("button", { name: "Save changes" }).click();
  const scope = page.getByRole("alertdialog");
  await expect(scope.getByRole("button", { name: "Only this one" })).toBeVisible();
  await scope.getByRole("button", { name: "This and future ones" }).click();
  await expect(upcoming.filter({ hasText: "120.00" })).toHaveCount(3);

  // Delete only the last one.
  await upcoming.last().click();
  await sheet.getByRole("button", { name: "Delete" }).click();
  await page.getByRole("alertdialog").getByRole("button", { name: "Only this one" }).click();
  await expect(upcoming).toHaveCount(2);

  // Back on the wallet, the balance still only counts today's 100.00.
  await page.getByRole("link", { name: "Transactions" }).click();
  await expect(page.getByRole("link", { name: "Transactions" })).toHaveAttribute(
    "aria-current",
    "page",
  );
  await expect(page.getByText("+900.00 USD").first()).toBeVisible();
  await expect(page.getByRole("button", { name: new RegExp(note) })).toHaveCount(1);
});
