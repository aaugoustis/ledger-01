/// <reference path="./.sst/platform/config.d.ts" />

// Serverless AWS deployment (SST 4). Deploys run from GitHub Actions (.github/workflows/deploy.yml);
// see "Deploy to AWS" in the README.
//   CloudFront Router ─┬─ /api/*, /auth/*, /mcp, /oauth/*, /.well-known/oauth-*
//                      │     ─► API Gateway (HTTP API) ─► Go Lambda (backend/cmd/lambda)
//                      └─ everything else ─► Next.js on Lambda via OpenNext (apps/web)
// The Go handler is the same one `task dev` serves locally; API_BASE_PATH=/api strips the prefix.
// Postgres is external (Neon): DatabaseUrl is the pooled URL, in the same region as AWS.

export default $config({
  app(input) {
    return {
      name: "ledger-01",
      removal: input?.stage === "production" ? "retain" : "remove",
      protect: input?.stage === "production",
      home: "aws",
      providers: { aws: { region: process.env.AWS_REGION ?? "sa-east-1" } },
    };
  },
  async run() {
    // SST's Go runtime can only build a module's root package, and every entrypoint here lives
    // under backend/cmd. Build all three bootstraps the usual way and ship them prebuilt.
    const { execSync } = await import("node:child_process");
    execSync("task build:lambda", { cwd: $cli.paths.root, stdio: "inherit" });

    const googleClientId = new sst.Secret("GoogleClientId");
    const googleClientSecret = new sst.Secret("GoogleClientSecret");
    const allowedEmails = new sst.Secret("AllowedEmails");
    const databaseUrl = new sst.Secret("DatabaseUrl");
    // Migrations need a session, so they skip Neon's transaction-mode pooler (-pooler host).
    // Other Postgres hosts have no such suffix and get the same URL.
    const databaseDirectUrl = databaseUrl.value.apply((url) => url.replace("-pooler.", "."));

    const router = new sst.aws.Router("Router");

    const env = {
      APP_ENV: "production",
      PUBLIC_URL: router.url,
      API_BASE_PATH: "/api",
      DATABASE_URL: databaseUrl.value,
      // Per Lambda instance; Neon's pooler fans the instances in.
      DB_MAX_CONNS: "2",
      GOOGLE_CLIENT_ID: googleClientId.value,
      GOOGLE_CLIENT_SECRET: googleClientSecret.value,
      ALLOWED_EMAILS: allowedEmails.value,
    };
    const goFunction = (name: "lambda" | "migrate" | "jobs") => ({
      runtime: "provided.al2023" as const,
      architecture: "arm64" as const,
      bundle: `backend/dist/${name}`,
      handler: "bootstrap",
      environment: env,
    });

    // Runs on every deploy (the input changes each time); a failing migration fails the deploy.
    // Migrations roll out next to the new API code, so keep them backward-compatible.
    const migrate = new sst.aws.Function("Migrate", {
      ...goFunction("migrate"),
      timeout: "5 minutes",
      environment: { ...env, DATABASE_URL: databaseDirectUrl },
    });
    new aws.lambda.Invocation("MigrateOnDeploy", {
      functionName: migrate.name,
      input: JSON.stringify({ deployedAt: Date.now() }),
    });

    const api = new sst.aws.ApiGatewayV2("Api");
    api.route("$default", { ...goFunction("lambda"), memory: "512 MB", timeout: "15 seconds" });

    // Scheduled jobs (backend/cmd/jobs): tops up recurring transactions and purges expired
    // imports. Idempotent, so a daily run is plenty for a three-month horizon.
    new sst.aws.CronV2("Jobs", {
      schedule: "rate(1 day)",
      function: { ...goFunction("jobs"), timeout: "5 minutes" },
    });

    new sst.aws.Nextjs("Web", {
      path: "apps/web",
      router: { instance: router },
      // SST's default OpenNext predates Next 16.4.
      openNextVersion: "4.1.13",
      // Server components call the API directly instead of going back through CloudFront.
      environment: { API_INTERNAL_URL: api.url },
    });

    router.route("/api", api.url);
    router.route("/auth", api.url);
    // MCP for AI assistants, and the OAuth server they sign in through (unprefixed, like /auth).
    router.route("/mcp", api.url);
    router.route("/oauth", api.url);
    router.route("/.well-known/oauth-protected-resource", api.url);
    router.route("/.well-known/oauth-authorization-server", api.url);

    return { url: router.url, api: api.url };
  },
});
