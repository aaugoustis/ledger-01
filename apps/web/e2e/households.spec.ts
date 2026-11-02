import { type APIRequestContext, expect, test } from "@playwright/test";

// An owner and a partner of their own per project, so the projects don't share a household.
const ownerFor = (project: string) => `e2e-hh-owner-${project}@example.com`;
const partnerFor = (project: string) => `e2e-hh-partner-${project}@example.com`;

function localToday() {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

async function rpc(request: APIRequestContext, method: string, data: object) {
  const res = await request.post(`/api/ledger.v1.${method}`, { data });
  expect(res.ok(), `${method}: ${await res.text()}`).toBe(true);
  return res.json();
}

async function signIn(request: APIRequestContext, email: string, name: string) {
  const res = await request.post("/auth/dev/login", { data: { email, name } });
  expect(res.status()).toBe(204);
}

test("invite a partner, share wallets, filter by person, remove them", async ({
  page,
  browser,
}, info) => {
  const ownerEmail = ownerFor(info.project.name);
  const partnerEmail = partnerFor(info.project.name);
  await signIn(page.request, ownerEmail, "Ada Owner");

  // Undo an earlier run: the partner leaves and stale invites go.
  const before = await rpc(page.request, "HouseholdService/ListMembers", {});
  for (const m of before.members ?? []) {
    if (m.role !== "owner")
      await rpc(page.request, "HouseholdService/RemoveMember", { userId: m.userId });
  }
  for (const inv of before.invites ?? []) {
    await rpc(page.request, "HouseholdService/CancelInvite", { email: inv.email });
  }

  const tag = Date.now() % 100000;
  const joint = `E2E Joint ${tag}`;
  const { wallet } = await rpc(page.request, "WalletService/CreateWallet", {
    name: joint,
    currency: "USD",
  });

  // The owner invites the partner from Settings → Household.
  await page.goto("/settings/household");
  await page.getByLabel("Invite by email").fill(partnerEmail);
  await page.getByRole("button", { name: "Invite" }).click();
  await expect(page.getByText(/They join when they next sign in/)).toBeVisible();
  await expect(page.getByText("joins on their next sign-in", { exact: false })).toBeVisible();

  // The partner signs in (in a browser of their own) and lands in the shared household.
  const partnerContext = await browser.newContext({ ...info.project.use });
  const partner = await partnerContext.newPage();
  await signIn(partner.request, partnerEmail, "Pat Partner");
  await partner.goto("/wallets");
  await expect(partner.getByText(joint).first()).toBeVisible();
  await partner.goto("/settings/household");
  await expect(partner.getByText("Only the owner can change these.")).toBeVisible();
  await expect(partner.getByLabel("Invite by email")).toHaveCount(0);

  // Each enters a transaction in the joint wallet.
  const cats = await rpc(page.request, "CategoryService/ListCategories", {});
  const category = cats.categories.find(
    (c: { kind: string }) => c.kind === "CATEGORY_KIND_EXPENSE",
  );
  for (const [request, note] of [
    [page.request, `E2E owner ${tag}`],
    [partner.request, `E2E partner ${tag}`],
  ] as const) {
    await rpc(request, "TransactionService/CreateTransaction", {
      walletId: wallet.id,
      kind: "TRANSACTION_KIND_EXPENSE",
      categoryId: category.id,
      amountMinor: "700",
      occurredOn: localToday(),
      note,
      today: localToday(),
    });
  }

  // The owner filters the wallet down to what the partner entered.
  await page.goto(`/wallets/${wallet.id}`);
  await expect(page.getByText(`E2E owner ${tag}`)).toBeVisible();
  await expect(page.getByText(`E2E partner ${tag}`)).toBeVisible();
  await page.getByRole("button", { name: /^Filters/ }).click();
  await page.getByRole("group", { name: "Entered by" }).getByText("Pat Partner").click();
  await page.keyboard.press("Escape");
  await expect(page.getByText(`E2E owner ${tag}`)).toHaveCount(0);
  await expect(page.getByText(`E2E partner ${tag}`)).toBeVisible();

  // Removing the partner takes the shared wallets away from them.
  await page.goto("/settings/household");
  await page.getByRole("button", { name: "Remove Pat Partner" }).click();
  await page.getByRole("alertdialog").getByRole("button", { name: "Remove" }).click();
  await expect(page.getByText("Removed Pat Partner")).toBeVisible();
  await partner.goto("/wallets");
  await expect(partner.getByText(joint)).toHaveCount(0);
  await partnerContext.close();
});
