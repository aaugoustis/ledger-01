# FxConverter

Three linked inputs for a cross-currency transfer: amount sent, amount received and the rate. Editing one recomputes the least recently edited of the other two.

Provide `from` and `to` (`{code, minorUnits}`), a controlled `value` (start from `fx.initialFxState()`) and `onChange`. Read the result with `fx.fxResult(value, {fromMinorUnits, toMinorUnits})`: integer minor units for both sides plus the rate at 10 decimals, or null while incomplete.

- The rule: `to = from × rate`, `from = to ÷ rate`, `rate = to ÷ from`. The field touched longest ago is overwritten by the other two. Before anything is touched, typing both amounts derives the rate, the usual case when copying a bank statement.
- The field that will be recomputed next shows a small `auto` mark in `hud-mark`.
- Rate means target units per one source unit, labelled "Rate (UYU per USD)". Computed rates show 6 decimals and keep 10 for storage.
- Amounts round half-even to each currency's decimals.
- Same-currency transfers show one MoneyInput instead.
- Three columns from 32rem of its own width (a container query), stacked below that, so it fits sheets and cards at any viewport.
