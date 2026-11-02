# Ledger/01

Self-hostable personal/household finance app (a multi-currency ledger). Monorepo: Go API, Next.js web
app, shared UI kit, protobuf contract. See README.md for setup.

## Layout

- `proto/ledger/v1/*.proto` — the API contract. Change it first, then `task gen`.
- `backend/` — Go module `github.com/franco-roura/ledger-01/backend`
  - `cmd/api` local server, `cmd/lambda` AWS Lambda entrypoint, `cmd/migrate` goose runner, `cmd/jobs`
    scheduled work (recurring top-up; `task jobs`, an EventBridge cron on AWS)
  - `internal/server` builds the single `http.Handler` both entrypoints serve
  - `internal/rpc` Connect handlers (thin), `internal/ledger` domain services, `internal/auth` Google sign-in + sessions
  - `internal/oauth` OAuth 2.1 server for MCP clients (`/oauth/*`, `/.well-known/oauth-*`, bearer
    tokens), `internal/mcp` read-only MCP tools on `POST /mcp` (stateless, thin over `ledger`)
  - `internal/importer` bank-statement parsing and matching, bank-agnostic (roles: purchase, discount,
    refund); each bank's wording lives in `internal/importer/banks/<country>/<bank>.go`, never in the core
  - `internal/db/migrations` goose SQL, `internal/db/queries` sqlc SQL, `internal/db/dbq` generated (sqlc)
  - `gen/` generated protobuf + Connect code
- `packages/api` — generated TS client (`src/gen`) + transports
- `packages/ui` — component kit: shadcn components themed with Module/01 tokens
- `apps/web` — Next.js App Router app (SSR), Playwright tests in `e2e/`
- `sst.config.ts` — AWS deployment (SST 4; Go functions ship prebuilt from `task build:lambda`),
  deployed by hand from `.github/workflows/deploy.yml`; `infra/` holds that workflow's AWS role

## Commands

`task setup`, `task db:up`, `task migrate`, `task dev`, `task gen`, `task test`, `task e2e`, `task lint`,
`task build:lambda`. Run `task --list` for everything. `bun x changeset` adds a changeset.

## Rules

- **Every PR that changes a package adds a changeset.** The packages are `apps/web`, `packages/*` and
  `backend` (which has a stub `package.json` for this). The app has one semver version (a Changesets
  `fixed` group), and CONTRIBUTING.md defines major, minor and patch. Use `bun x changeset --empty`
  when nothing user-visible changes. Never bump versions or edit CHANGELOG.md by hand: the Release
  workflow's "Version packages" PR does it, and tags are plain `vX.Y.Z`.
- **Never edit generated code** (`backend/gen`, `backend/internal/db/dbq`, `packages/api/src/gen`). Edit
  the `.proto` / `.sql` source and run `task gen`.
- **Money is integer minor units** (`int64` / `bigint`) plus an ISO 4217 code. Never floats. Transfer
  rates are decimal strings in the API and `numeric` in Postgres.
- **Every row belongs to a household.** Every query filters by the caller's `household_id`, which comes
  from the session, never from the request.
- **Keep the API Lambda-safe:** stateless (sessions live in Postgres), config only from env vars
  (`internal/config`), unary RPCs only (API Gateway buffers responses), no background goroutines in
  the API process (scheduled work goes in `cmd/jobs`), migrations only via `cmd/migrate`. Only the
  `cmd/lambda`, `cmd/migrate` and `cmd/jobs` entrypoints may import AWS packages.
- **Styling only through `packages/ui` tokens** (Tailwind theme / CSS variables generated from
  `packages/ui/tokens/tokens.json`). No raw hex values in app code. Icons: Lucide, `strokeWidth={1.25}`.
  Light theme only. Text on `signal` orange is always `ink`, never white.
- **Forms flow by themselves** when built from kit parts: `Dialog`/`Sheet` focus the first field on
  open (and open it if it is an empty `Combobox`); `Select`, `Combobox` and `DatePicker` move focus to the next field after a pick (`MultiCombobox` stays open while toggling); Tab follows
  DOM order. So lay fields out in fill order, use kit controls for choices (never a hand-rolled
  Popover picker), don't add `autoFocus` inside dialogs, and validate with `fault(id)` + the
  Field's `error`. Keep phone input text at 16px or more, or mobile browsers zoom in on focus. Helpers:
  `packages/ui/src/lib/form-flow.ts`, `lib/fault.ts`.
- UI copy is English, sentence case, plain words. HUD codes (`W-01`, `S-02`) are decoration, not headings.
- **Today comes from the client.** Rows dated after the caller's `today` are upcoming (recurring
  occurrences are pre-created as real rows): they never count in balances, reports, budgets or rates,
  and lists show them only in the Future view. Requests that depend on it take a `today` field.
- Expenses and income are always in their wallet's currency. Only transfers cross currencies, entered
  with the three-field converter (from amount, to amount, rate; the least recently edited field is
  recomputed).
- Examples, tests, fixtures and docs use made-up bank names, people and amounts, never real account
  data.
- Tests: Go integration tests use testcontainers (Docker must be running). Web unit tests use
  `bun test`. E2E uses the dev-only login (`APP_ENV=development` + `AUTH_DEV_LOGIN=1`).

## Design system

The UI follows the **Module/01** design system. Its source tokens are vendored in
`packages/ui/tokens/tokens.json` (run `bun run --cwd packages/ui tokens` after editing). When you add a
token or component to `packages/ui`, document it in the design-system artifact in the same change
(maintainers only): `task ds:bundle` rebuilds `packages/ui/artifact/project/` (bundle, styles, types,
tokens); component guidelines and previews live next to it under `components/<Name>/`.
