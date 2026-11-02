import { defineConfig } from "@playwright/test";

// Docs screenshots (docs-shots/), not tests: `task docs:screenshots`. Same stack as e2e, plus a
// second API on :8081 that lets the made-up docs account sign in. Start the database first.
export default defineConfig({
  testDir: "./docs-shots",
  reporter: "list",
  timeout: 300_000,
  expect: { timeout: 20_000 },
  use: { baseURL: "http://localhost:3000" },
  webServer: [
    {
      command: "go run ./cmd/api",
      cwd: "../../backend",
      url: "http://127.0.0.1:8080/healthz",
      reuseExistingServer: true,
      timeout: 120_000,
    },
    {
      command: "go run ./cmd/api",
      cwd: "../../backend",
      url: "http://127.0.0.1:8081/healthz",
      env: {
        ...(process.env as Record<string, string>),
        API_ADDR: "127.0.0.1:8081",
        APP_ENV: "development",
        AUTH_DEV_LOGIN: "1",
        ALLOWED_EMAILS: "docs@example.com",
      },
      reuseExistingServer: false,
      timeout: 120_000,
    },
    {
      command: "bun run dev",
      url: "http://localhost:3000/login",
      reuseExistingServer: true,
      timeout: 120_000,
    },
  ],
});
