# BudgetMeter

How much of a budget is spent: a sunken `shell-recess` track (`shadow-press`) filled to the share spent, and a thin `ink` tick with a diamond head where today falls in the period.

Provide `spent` and `budget` (minor units, BigInt), `label` (its accessible name, e.g. "Food budget") and optionally `elapsed` (0–1, the share of the period that has passed; omit it outside the period to hide the tick). `size` is `md` (12px track) or `sm` (8px, for history rows).

- The fill is `signal` while within budget and turns `danger` once spending passes it; it stops at the end of the track.
- Fill ahead of the tick means spending is ahead of the calendar.
- It's a `meter` for assistive tech, valued in percent spent.
- Always pair it with the figures in text ("12.34 USD of 100.00 USD spent", "Over by 2.34 USD"); color is never the only cue.
- Put the period's dates under it, and a "◆ Today" caption under the tick.
