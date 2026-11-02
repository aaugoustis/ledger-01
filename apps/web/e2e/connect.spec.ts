import { createHash, randomBytes } from "node:crypto";
import { expect, type Page, test } from "@playwright/test";

const email = process.env.E2E_EMAIL ?? "e2e@example.com";
// A made-up MCP client; the browser never really leaves for it (see catchRedirect).
const redirectUri = "https://client.example.com/callback";
const verifier = randomBytes(32).toString("base64url");
const challenge = createHash("sha256").update(verifier).digest("base64url");

async function signIn(page: Page) {
  const res = await page.request.post("/auth/dev/login", { data: { email, name: "E2E Tester" } });
  expect(res.status()).toBe(204);
}

/** Answers the client's redirect locally, so the test can read the code from the URL. */
async function catchRedirect(page: Page) {
  await page.route(`${redirectUri}**`, (route) =>
    route.fulfill({ status: 200, contentType: "text/plain", body: "back at the client" }),
  );
}

function authorizeUrl(clientId: string, state: string) {
  return `/oauth/authorize?${new URLSearchParams({
    response_type: "code",
    client_id: clientId,
    redirect_uri: redirectUri,
    state,
    code_challenge: challenge,
    code_challenge_method: "S256",
    resource: "http://localhost:3000/mcp",
  })}`;
}

test("connect an MCP client: sign in, allow, use, disconnect", async ({ page }) => {
  const clientName = `E2E Assistant ${Math.random().toString(36).slice(2, 7)}`;

  // Discovery and registration go through the web origin, as a real client sees them.
  const meta = await page.request.get("/.well-known/oauth-authorization-server");
  expect(meta.ok()).toBe(true);
  expect((await meta.json()).registration_endpoint).toBe("http://localhost:3000/oauth/register");
  const reg = await page.request.post("/oauth/register", {
    data: { client_name: clientName, redirect_uris: [redirectUri] },
  });
  expect(reg.status()).toBe(201);
  const clientId: string = (await reg.json()).client_id;

  // Signed out, authorizing goes through the login page first.
  await page.goto(authorizeUrl(clientId, "s1"));
  await expect(page).toHaveURL(/\/login\?next=/);
  await expect(page.getByRole("link", { name: "Continue with Google" })).toBeVisible();

  await signIn(page);
  await catchRedirect(page);

  // Deny sends the browser back with access_denied.
  await page.goto(authorizeUrl(clientId, "s1"));
  await expect(page.getByRole("heading", { name: `Connect ${clientName}?` })).toBeVisible();
  await expect(page.getByText("client.example.com")).toBeVisible();
  await page.getByRole("button", { name: "Deny" }).click();
  await page.waitForURL(`${redirectUri}**`);
  expect(new URL(page.url()).searchParams.get("error")).toBe("access_denied");

  // Allow returns a code for the client to exchange.
  await page.goto(authorizeUrl(clientId, "s2"));
  await expect(page).toHaveURL(/\/connect\?request=/);
  const consentUrl = page.url();
  await page.getByRole("button", { name: "Allow read access" }).click();
  await page.waitForURL(`${redirectUri}**`);
  const back = new URL(page.url()).searchParams;
  expect(back.get("state")).toBe("s2");
  const token = await page.request.post("/oauth/token", {
    form: {
      grant_type: "authorization_code",
      client_id: clientId,
      code: back.get("code") ?? "",
      redirect_uri: redirectUri,
      code_verifier: verifier,
    },
  });
  expect(token.ok()).toBe(true);
  const access: string = (await token.json()).access_token;

  const mcp = (bearer: string) =>
    page.request.post("/mcp", {
      headers: {
        Authorization: `Bearer ${bearer}`,
        Accept: "application/json, text/event-stream",
      },
      data: { jsonrpc: "2.0", id: 1, method: "tools/list", params: {} },
    });
  const listed = await mcp(access);
  expect(listed.ok()).toBe(true);
  const names = (await listed.json()).result.tools.map((t: { name: string }) => t.name);
  expect(names).toContain("search_transactions");

  // The answered request can't be used again.
  await page.goto(consentUrl);
  await expect(page.getByRole("heading", { name: "This link has expired" })).toBeVisible();

  // Settings lists the app and disconnects it.
  await page.goto("/settings/connected-apps");
  await expect(page.getByText(clientName)).toBeVisible();
  await expect(page.getByText(/\/mcp$/)).toBeVisible();
  await page.getByRole("button", { name: `Disconnect ${clientName}` }).click();
  await expect(page.getByText(clientName)).toHaveCount(0);
  expect((await mcp(access)).status()).toBe(401);
});
