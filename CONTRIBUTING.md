# Contributing

Thanks for helping out. Bug reports, fixes and new bank definitions are all welcome.

## Getting started

Follow the [Quickstart](docs/development.md#quickstart). You need Go, Bun, Node.js, Docker and
[Task](https://taskfile.dev). To work without a Google OAuth client, set `AUTH_DEV_LOGIN=1` in `.env`
(development only) and sign in with any email listed in `ALLOWED_EMAILS`. The response sets the
session cookie:

```bash
curl -i -X POST localhost:3000/auth/dev/login -H 'content-type: application/json' -d '{"email":"you@example.com"}'
```

## Before you open a pull request

- Run `task lint` and `task test`. The Go tests start Postgres with testcontainers, so Docker must be
  running.
- Run `task e2e` when you change the web app.
- If you changed a `.proto` or `.sql` file, run `task gen` and commit the generated code. CI fails
  when generated code is out of date.
- Add a changeset (see below). CI fails when a changed package has none.
- Keep each pull request to one change, and say what changed and why.

## Versioning and changesets

Ledger/01 follows [semantic versioning](https://semver.org) with one version for the whole app (the
web app, the Go API and the shared packages). A version number tells someone who self-hosts what an
upgrade involves:

- **Major:** upgrading needs manual steps. Examples: a new or renamed environment variable or
  secret, a migration that can't run on its own, or a breaking change to the API contract or the
  deployment setup.
- **Minor:** new features. Upgrading is a plain redeploy.
- **Patch:** fixes.

Describe your change with `bun x changeset`. Pick the package you touched (`@ledger-01/backend` for
Go), choose the bump, and write one or two sentences for the release notes, addressed to someone who
uses the app. For changes nobody would notice (tests, refactors, CI), use `bun x changeset --empty`.

You don't edit versions or changelogs by hand. The Release workflow collects pending changesets into a
"Version packages" pull request. Merging it bumps every package, updates the changelogs, tags
`vX.Y.Z` and publishes the GitHub release.

## Ground rules

- **The API contract comes first.** Edit `proto/ledger/v1/*.proto`, then run `task gen`. Never edit
  generated code (`backend/gen`, `backend/internal/db/dbq`, `packages/api/src/gen`).
- **Money is integer minor units** plus an ISO 4217 currency code. Never floats.
- **Every query filters by the caller's household**, taken from the session, never from the request.
- **The API stays Lambda-safe:**
  - it's stateless and reads its config only from environment variables;
  - every RPC is unary, and there are no background goroutines;
  - scheduled work lives in `cmd/jobs`.
- **The UI is styled only through `packages/ui` tokens**, and forms are built from the kit's parts.
  Copy is English, in sentence case.
- **Tests, fixtures and docs use made-up data:** banks, people, merchants and amounts. Never copy real
  statement lines.

[CLAUDE.md](CLAUDE.md) has the full set of rules. It doubles as the guide for coding agents.

## Adding your bank

Ledger/01 recognises discounts and refunds in bank statements, but only for banks it knows. A bank is
one small Go file with a few patterns and a test:

1. Note how your statements word purchases, discounts and refunds.
2. Add `backend/internal/importer/banks/<country>/<bank>.go`, modelled on the Itaú one.
3. Register it in `banks.All()`.
4. Add a test with made-up data.
5. Run `task test` and `task lint`, then open a pull request.

The full guide is in [backend/internal/importer/banks/README.md](backend/internal/importer/banks/README.md).

## Bugs and security issues

Open an issue for bugs, with the steps to reproduce them. For anything security-related, follow
[SECURITY.md](SECURITY.md) instead of opening an issue.
