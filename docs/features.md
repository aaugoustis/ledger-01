[← README](../README.md) · **Features** · [Getting started](getting-started.md) · [Your data](your-data.md) · [AI assistants](ai-assistants.md) · [Self-hosting](self-hosting.md) · [Architecture](architecture.md) · [Development](development.md)

# Features

Everything Ledger/01 does today, shown with a made-up household.

**On this page:** [Wallets](#wallets) · [Transactions](#transactions) ·
[Transfers between currencies](#transfers-between-currencies) · [Dashboards](#dashboards) ·
[Budgets](#budgets) · [Recurring transactions](#recurring-transactions-and-the-future-view) ·
[On your phone](#on-your-phone) · [Sign-in and households](#sign-in-and-households) ·
[More](#more) · [Roadmap and releases](#roadmap-and-releases)

## Wallets

A wallet is any place money lives: a card, a savings account, a peso account, the cash in your
pocket. Each one has its own currency and opening balance. The wallets page totals them in your base
currency.

![The wallets page with four wallets and their balances](screenshots/wallets.png)

## Transactions

Every expense and income has a category, an optional note and optional labels. Labels cut across
categories: "Shared" marks costs you split, and "Vacation" gathers a trip's flights and hotels. Each
wallet lists its transactions by day, with a search over notes and filters by amount, category and
label.

![A wallet's transactions grouped by day, with categories, notes and a label](screenshots/wallet.png)

> [!TIP]
> The new transaction sheet opens on its first field, and every pick moves you to the next one. A
> typical expense takes four taps on a phone once the wallet is open, and no more from the
> dashboard: there the sheet starts in the wallet you last added to, and a Wallet field below the
> amount picks another.

## Transfers between currencies

Expenses and income always use their wallet's currency. Only transfers cross currencies, and they use
a three-field converter. Type any two of **sent**, **received** and **rate**, and the field you
touched least recently is worked out. To move money at the rate your bank gave you, type the two
amounts from the statement and the rate fills itself in.

![The transfer form: 250.00 USD sent, 10,125.00 UYU received, rate 40.5 filled in](screenshots/transfer.png)

## Dashboards

The dashboard shows the household's money in one place, converted to your base currency:

- the total balance, and the change, expenses and income for the period;
- the balance over time;
- cash flow by day, week or month;
- spending and income by category, with a drill-down into each one.

Filter by wallet, category, note or amount. Each wallet has an overview page with the same charts.

![The dashboard by week: balance over three months and weekly income and expenses](screenshots/dashboard.png)

![A wallet's overview page](screenshots/wallet-overview.png)

## Budgets

A budget covers any set of categories and wallets. It can be one-off or repeat daily, weekly, every two
weeks, monthly or yearly. Each card shows what's spent, what's left, and how much you can spend per
day for the rest of the period. When you go over, it says by how much.

![Three monthly budgets: two on track, one over by 207.33 USD](screenshots/budgets.png)

## Recurring transactions and the Future view

Rent, subscriptions, a salary: set how often it repeats, choose from daily to yearly, and the
upcoming ones are created ahead of time. They stay out of balances, reports and budgets until their
date arrives. Until then they live in the **Future** view, where you can edit one occurrence or all
the ones after it.

![The Future view listing upcoming rent and subscription payments](screenshots/future.png)

## On your phone

Ledger/01 installs as an app on phones (a PWA): add it to your home screen from the browser. The
layout is built for one-handed use, with a pinned **New transaction** button and inputs large enough
that the browser doesn't zoom in. Press and hold the app's icon for a **New transaction** shortcut.

<p align="center">
  <img src="screenshots/mobile-dashboard.png" width="260" alt="The dashboard on a phone">
  <img src="screenshots/mobile-wallet.png" width="260" alt="A wallet on a phone">
  <img src="screenshots/mobile-new-transaction.png" width="260" alt="The new transaction sheet on a phone">
</p>

## Sign-in and households

People sign in with Google, and only the accounts you list in `ALLOWED_EMAILS` get in. Everything
belongs to a household, and every query is scoped to the household of the signed-in person.

## More

- **Bank statement import** with column mapping, duplicate detection, and discount and refund
  matching. See [Your data](your-data.md#import-bank-statements).
- **Backups and CSV export.** See [Your data](your-data.md#backups).
- **AI assistants** with read-only access over MCP. See [AI assistants](ai-assistants.md).

## Roadmap and releases

Next up: inviting household members, and more LLM-assisted features.

Releases follow semantic versioning, and each one's notes are on the
[Releases page](https://github.com/franco-roura/ledger-01/releases). A major version means the upgrade
needs manual steps; a minor or patch release is a plain redeploy.

---

[← README](../README.md) · Next: [Getting started →](getting-started.md)
