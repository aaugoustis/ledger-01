# @ledger-01/web

## 1.0.1

### Patch Changes

- Updated dependencies []:
  - @ledger-01/api@1.0.1
  - @ledger-01/ui@1.0.1

## 1.0.0

Initial public release of Ledger/01, a self-hostable personal and household finance app.

- Wallets in any currency, with categories, labels and notes on every transaction.
- Transfers between currencies: enter any two of the amount sent, the amount received and the rate,
  and the third is worked out.
- Dashboards: balance over time, cash flow and spending by category, filtered by wallet, category,
  note or amount.
- Budgets for any set of categories and wallets, one-off or repeating.
- Recurring transactions, with upcoming ones in a Future view that stay out of balances until their
  date.
- Bank statement import (CSV, XLSX, XLS) with column mapping and duplicate detection. Discounts and
  refunds are matched to the purchases they belong to.
- Backups: download the whole household as one file, restore it here or on another instance, and
  export transactions as CSV.
- Shared households: invite people by email, see and edit the same wallets, and filter
  transactions by who entered them.
- Ask Claude about your money: read-only MCP tools on `/mcp`, connected over OAuth with your
  Google sign-in, and a Connected apps page to disconnect them.
- Installs on phones and desktops as a PWA; the sign-in page offers to install it.
- Google sign-in for the people you allow.
- Deploys to your own AWS account with SST: the Go API on Lambda behind API Gateway, the web app
  through OpenNext, Postgres on Neon.
