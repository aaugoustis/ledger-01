// Captures the screenshots in docs/screenshots from a made-up household (demo-household.ts).
// Run with `task docs:screenshots`. It restores the demo into its own household (docs@example.com),
// signing in through a second API that allows that account; sessions live in Postgres, so the
// session works on the web server too, whether `task dev` started it or this run did.
import { readFileSync } from "node:fs";
import { mkdir } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { gunzipSync } from "node:zlib";
import {
  type Browser,
  type BrowserContext,
  devices,
  expect,
  type Page,
  request,
  test,
} from "@playwright/test";
import { type BackupDoc, demoBackup } from "./demo-household";

const out = fileURLToPath(new URL("../../../docs/screenshots/", import.meta.url));
const statement = fileURLToPath(
  new URL("../../../backend/internal/importer/testdata/northbank.csv", import.meta.url),
);
const tokens = JSON.parse(
  readFileSync(new URL("../../../packages/ui/tokens/tokens.json", import.meta.url), "utf8"),
) as { color: { tokens: { name: string; value: string }[] } };
const color = (name: string) => tokens.color.tokens.find((t) => t.name === name)?.value ?? "";

const DOCS_EMAIL = "docs@example.com";
const DOCS_API = "http://127.0.0.1:8081";
const BASE = "http://localhost:3000";

const localToday = () => {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
};

async function signedInContext(browser: Browser, options: Parameters<Browser["newContext"]>[0]) {
  const api = await request.newContext();
  const res = await api.post(`${DOCS_API}/auth/dev/login`, {
    data: { email: DOCS_EMAIL, name: "Alex Rivera" },
  });
  expect(res.status()).toBe(204);
  const token = (await api.storageState()).cookies.find((c) => c.name === "ledger_session");
  if (!token) throw new Error("the dev login set no session cookie");
  await api.dispose();
  const context = await browser.newContext({ ...options, baseURL: BASE });
  await context.addCookies([
    { name: token.name, value: token.value, domain: "localhost", path: "/", httpOnly: true },
  ]);
  return context;
}

// Puts a capture on the studio backdrop, the way Module/01 shows media.
async function frame(context: BrowserContext, shot: Buffer, cssWidth: number, name: string) {
  const page = await context.browser()?.newPage({ deviceScaleFactor: 2 });
  if (!page) throw new Error("no browser");
  const src = `data:image/png;base64,${shot.toString("base64")}`;
  await page.setContent(`<body style="margin:0">
    <div id="f" style="display:inline-block;padding:48px;background:${color("studio")}">
      <img src="${src}" style="display:block;width:${cssWidth}px;border-radius:14px;
        box-shadow:0 4px 10px rgba(28,30,33,0.06),0 24px 48px rgba(28,30,33,0.10)">
    </div></body>`);
  await page.locator("#f").screenshot({ path: `${out}${name}.png` });
  await page.close();
}

async function capture(page: Page, name: string, cssWidth: number) {
  await page.waitForLoadState("networkidle");
  // The Next.js dev-mode badge isn't part of the app.
  await page.evaluate(() => {
    for (const el of document.querySelectorAll("nextjs-portal")) el.remove();
  });
  await page.waitForTimeout(600); // chart and sheet animations
  await frame(page.context(), await page.screenshot(), cssWidth, name);
}

test("docs screenshots", async ({ browser }) => {
  test.setTimeout(300_000);
  await mkdir(out, { recursive: true });
  const desktop = await signedInContext(browser, {
    viewport: { width: 1280, height: 800 },
    deviceScaleFactor: 2,
  });
  const page = await desktop.newPage();

  // Restore the demo over whatever the docs household holds, starting from its own backup so the
  // default categories and the schema version are the server's.
  await page.goto("/settings/data");
  const [download] = await Promise.all([
    page.waitForEvent("download"),
    page.getByRole("button", { name: "Download backup" }).click(),
  ]);
  const base = JSON.parse(gunzipSync(readFileSync(await download.path())).toString()) as BackupDoc;
  const demo = demoBackup(base, localToday());
  await page.getByLabel("Backup file").setInputFiles({
    name: "maple-street.json",
    mimeType: "application/json",
    buffer: Buffer.from(JSON.stringify(demo)),
  });
  await expect(page.getByRole("term").filter({ hasText: /^Transactions$/ })).toBeVisible();
  await capture(page, "settings-data", 1280);
  await page.getByRole("button", { name: "Restore backup" }).click();
  await page.getByRole("alertdialog").getByRole("button", { name: "Replace data" }).click();
  await expect(page.getByText(/^Restored \d+ transactions$/)).toBeVisible();

  const wallets = await (
    await page.request.post("/api/ledger.v1.WalletService/ListWallets", { data: {} })
  ).json();
  const card = wallets.wallets.find((w: { name: string }) => w.name === "Everyday card").id;

  await page.goto("/dashboard");
  await expect(page.getByText("Total balance")).toBeVisible();
  // A few months by week shows the salary-and-spending rhythm; tall enough for the charts.
  await page.getByText("Weeks", { exact: true }).click();
  await page.setViewportSize({ width: 1280, height: 1180 });
  await capture(page, "dashboard", 1280);
  await page.setViewportSize({ width: 1280, height: 800 });

  await page.goto("/wallets");
  await capture(page, "wallets", 1280);

  await page.goto(`/wallets/${card}`);
  await expect(page.getByRole("heading", { name: "Everyday card" })).toBeVisible();
  await capture(page, "wallet", 1280);

  await page.goto(`/wallets/${card}/overview`);
  await capture(page, "wallet-overview", 1280);

  await page.goto(`/wallets/${card}`);
  await page.getByRole("button", { name: "New transaction" }).first().click();
  const sheet = page.getByRole("dialog");
  // The sheet opens with its category list open; Escape closes just the list.
  await expect(page.getByPlaceholder("Search categories…")).toBeFocused();
  await page.keyboard.press("Escape");
  await sheet.getByRole("tab", { name: "Transfer" }).click();
  await sheet.getByLabel("To wallet").click();
  await page.getByRole("option", { name: /Peso account/ }).click();
  await sheet.getByLabel("Sent (USD)").fill("250.00");
  await sheet.getByLabel("Received (UYU)").fill("10,125.00");
  await expect(sheet.getByLabel("Rate (UYU per USD)")).not.toHaveValue("");
  await capture(page, "transfer", 1280);
  await page.keyboard.press("Escape");

  await page.goto(`/wallets/${card}/import`);
  await page.getByLabel("Statement file").setInputFiles(statement);
  await expect(page.getByLabel("Date format")).toBeVisible();
  await capture(page, "import-columns", 1280);
  await page.getByRole("button", { name: "Review rows" }).click();
  await expect(page.getByText(/\d+ rows · \d+ ticked/)).toBeVisible();
  await capture(page, "import-review", 1280);

  await page.goto("/budgets");
  await capture(page, "budgets", 1280);

  await page.goto("/future");
  await capture(page, "future", 1280);

  const phone = await signedInContext(browser, { ...devices["Pixel 7"] });
  const mobile = await phone.newPage();
  await mobile.goto("/dashboard");
  await expect(mobile.getByText("Total balance")).toBeVisible();
  await capture(mobile, "mobile-dashboard", 412);
  await mobile.goto(`/wallets/${card}`);
  await capture(mobile, "mobile-wallet", 412);
  await mobile.getByRole("button", { name: "New transaction" }).last().click();
  await expect(mobile.getByPlaceholder("Search categories…")).toBeFocused();
  await mobile.keyboard.press("Escape");
  await capture(mobile, "mobile-new-transaction", 412);

  await desktop.close();
  await phone.close();
});
