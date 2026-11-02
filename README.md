<p align="center">
  <img src="docs/brand/icon.png" width="160" height="160" alt="Ledger/01 icon: a white molded device with a dark balance screen and an orange L/01 nameplate">
</p>

<h1 align="center">Ledger/01</h1>

<p align="center">
  <b>A self-hostable personal and household finance app.</b><br>
  Track every wallet, in any currency, on every device, on a server you own.<br>
  It costs close to nothing to run.
</p>

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/API-Go-1c1e21?style=flat-square&logo=go">
  <img alt="Next.js" src="https://img.shields.io/badge/web-Next.js-1c1e21?style=flat-square&logo=nextdotjs">
  <img alt="Self-hosted on AWS" src="https://img.shields.io/badge/self--hosted-AWS%20Lambda-ffd9c7?style=flat-square">
  <img alt="License: MIT OR Apache-2.0" src="https://img.shields.io/badge/license-MIT%20%2F%20Apache--2.0-53575e?style=flat-square">
</p>

<p align="center">
  <img src="docs/screenshots/dashboard.png" alt="The Ledger/01 dashboard: wallet balances, totals for the period, a balance chart and income and expenses by week">
</p>

## Contents

- [Why Ledger/01](#why-ledger01)
- [Everything a household ledger needs](#everything-a-household-ledger-needs)
- [Up and running in five minutes](#up-and-running-in-five-minutes)
- [Your bank statements, without the typing](#your-bank-statements-without-the-typing)
- [Ask your AI about your money](#ask-your-ai-about-your-money)
- [Host it yourself on AWS](#host-it-yourself-on-aws)
- [Serverless, so it costs cents](#serverless-so-it-costs-cents)
- [Open source, open contract](#open-source-open-contract)
- [Contributing](#contributing)
- [Security and license](#security-and-license)

## Why Ledger/01

<table>
  <tr>
    <td width="50%" valign="top">
      <b>You own your data</b><br>
      It runs in your AWS account, against your database. Download the whole ledger as one file
      whenever you like. <a href="docs/your-data.md">Your data →</a>
    </td>
    <td width="50%" valign="top">
      <b>One ledger, every device</b><br>
      Install it on your phone and laptop, and share one household with the people you allow.
      <a href="docs/features.md">Features →</a>
    </td>
  </tr>
  <tr>
    <td valign="top">
      <b>Drop in a statement, done</b><br>
      Drag your bank's CSV or Excel file onto the page. Duplicates, discounts and refunds are
      sorted out for you. <a href="docs/your-data.md#import-bank-statements">Importing →</a>
    </td>
    <td valign="top">
      <b>Costs cents a month</b><br>
      Everything scales to zero. A household's usage fits in AWS's and Neon's free tiers.
      <a href="docs/architecture.md#what-it-costs">Costs →</a>
    </td>
  </tr>
  <tr>
    <td valign="top">
      <b>Ask your AI</b><br>
      Connect Claude and ask "am I spending more on takeout than last month?". It can read,
      never change. <a href="docs/ai-assistants.md">AI assistants →</a>
    </td>
    <td valign="top">
      <b>Open and transparent</b><br>
      MIT or Apache-2.0. Every line, every API call and every release note is public.
      <a href="docs/development.md">Development →</a>
    </td>
  </tr>
</table>

## Everything a household ledger needs

Wallets in any currency, with categories, labels and notes on every transaction. Transfers between
currencies, where you type any two of sent, received and rate. Dashboards, budgets, and recurring
bills that show up in a Future view without touching today's balance.

**[Read more: the features tour →](docs/features.md)**

## Up and running in five minutes

Four commands start Postgres, the API and the web app on your machine, and a dev-only login skips
the Google setup. The guide then walks you through your first wallet, transaction, import and
budget.

**[Read more: getting started →](docs/getting-started.md)**

## Your bank statements, without the typing

Import CSV, XLSX and XLS statements with column mapping and duplicate detection. Discounts and refunds
are matched to the purchases they belong to. Backups restore on any instance, and CSV export opens
straight in a spreadsheet.

**[Read more: import, backup and export →](docs/your-data.md)**

## Ask your AI about your money

Ledger/01 is an MCP server. Add it to Claude as a connector, sign in, and ask questions about your own
numbers. Access is read-only, and you can disconnect it from Settings at any time.

**[Read more: AI assistants →](docs/ai-assistants.md)**

## Host it yourself on AWS

One GitHub Actions workflow deploys the whole app to your own AWS account with SST. There are no
long-lived keys: the workflow signs in to AWS with GitHub's OIDC, and your secrets stay in your
repository's `production` environment.

**[Read more: self-hosting →](docs/self-hosting.md)**

## Serverless, so it costs cents

The Go API and the Next.js app run on Lambda behind CloudFront, and Postgres runs on Neon, which
scales to zero. With nothing running, nothing is billed. A typical household's month comes to about
ten cents.

<p align="center">
  <a href="docs/architecture.md"><img src="docs/brand/architecture.png" alt="Deployment map: devices reach CloudFront, which routes to Next.js and API Gateway on Lambda; the Go API uses Neon Postgres and Google sign-in; GitHub Actions deploys" width="720"></a>
</p>

**[Read more: architecture and costs →](docs/architecture.md)**

## Open source, open contract

A Go API on Connect-RPC, a Next.js web app, and one protobuf contract that generates both clients.
Run `task test` for integration tests against a real Postgres, and Playwright for end-to-end tests.
Every release follows semantic versioning.

**[Read more: local development →](docs/development.md)**

## Contributing

Bug reports, fixes and new banks are all welcome. Adding your bank is one small Go file and a test.

**[Read more: CONTRIBUTING.md →](CONTRIBUTING.md)**

## Security and license

Report security issues privately as described in [SECURITY.md](SECURITY.md).

Dual-licensed under either of [MIT](LICENSE-MIT) or [Apache-2.0](LICENSE-APACHE), at your option.
