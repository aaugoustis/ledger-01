import { defineConfig, devices } from "@playwright/test";

// E2E runs against the real stack: Go API (dev login enabled) + Next dev server + the compose
// Postgres. Start the database first (`task db:up && task migrate`).
const apiEnv = {
  ...process.env,
  APP_ENV: "development",
  AUTH_DEV_LOGIN: "1",
  // data.spec.ts restores backups into households of its own, one per project.
  ALLOWED_EMAILS: [
    process.env.E2E_EMAIL ?? "e2e@example.com",
    "e2e-backup-desktop@example.com",
    "e2e-backup-mobile@example.com",
    // households.spec.ts: an owner and a partner per project.
    ...["desktop", "mobile"].flatMap((p) => [
      `e2e-hh-owner-${p}@example.com`,
      `e2e-hh-partner-${p}@example.com`,
    ]),
  ].join(","),
};

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? "github" : "list",
  // The Next dev server compiles each route on its first visit, which can take longer than the
  // defaults (5 s per assertion, 30 s per test) on a cold start. CI always starts cold.
  timeout: 120_000,
  expect: { timeout: 15_000 },
  use: {
    baseURL: "http://localhost:3000",
    trace: "retain-on-failure",
  },
  projects: [
    { name: "desktop", use: { ...devices["Desktop Chrome"] } },
    { name: "mobile", use: { ...devices["Pixel 7"] } },
  ],
  webServer: [
    {
      command: "go run ./cmd/api",
      cwd: "../../backend",
      url: "http://127.0.0.1:8080/healthz",
      env: apiEnv as Record<string, string>,
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
    },
    {
      command: "bun run dev",
      url: "http://localhost:3000/login",
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
    },
  ],
});
