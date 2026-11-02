# Security policy

Ledger/01 holds people's financial data, so security reports come first.

## Reporting a vulnerability

Report it privately on GitHub: open the repository's **Security** tab and choose **Report a
vulnerability**. Please don't open a public issue or pull request for it.

Include what you found, how to reproduce it and what an attacker could do with it. Expect an
acknowledgement within a week. This is a personal project maintained in spare time, so a fix may take
longer, but you'll hear how it's going until it ships. You'll be credited in the fix unless you'd
rather not be.

## Supported versions

Only the latest `main` gets fixes. Self-hosted instances should update to it.

## Scope

In scope: the Go API, the web app, and the deployment setup (`sst.config.ts`, `infra/`,
`.github/workflows/`).

Out of scope: the setup of a particular self-hosted instance (its AWS account, DNS or Google OAuth
client), volume-based denial of service, and anything that needs an already compromised device or
browser.

## How the app protects data

- Sign-in is Google OAuth (authorization code with PKCE, state and nonce). Only verified emails on the
  `ALLOWED_EMAILS` list can sign in.
- The email-only login used by end-to-end tests exists only when `APP_ENV=development` and
  `AUTH_DEV_LOGIN=1`.
- Sessions are random tokens stored as SHA-256 hashes. The cookie is HttpOnly, SameSite=Lax and
  Secure over HTTPS.
- Every row belongs to a household, and every query filters by the household from the session, never
  from the request.
- Configuration comes from environment variables, and deploys reach AWS through OIDC, so the
  repository holds no secrets or long-lived keys.
