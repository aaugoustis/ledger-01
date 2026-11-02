# Amount

A money figure in tabular Space Mono, signed and colored by direction: `+1,234.56 USD`, `−42.05 USD`.

Provide `minor` (an integer amount in minor units, as a BigInt), `currency` (ISO 4217) and `minorUnits` (decimal places). Optional `sign` (`always` default, `auto`, `never`), `code` (show the currency code, default true), `size` (`sm` = `hud-num`, `md` = `amount-md` default, `lg` = `amount-lg`, `xl` = `amount-xl`) and `tone` (`auto`, `plain` ink, `muted` ink-soft).

- Positive is `income-ink`, negative `expense-ink`, zero `ink`. The sign is always printed, so color is never the only cue.
- Negatives use a true minus (−), which lines up with + in mono.
- Never format money from floats; amounts are integer minor units end to end.
- Use `muted` for a secondary figure, like the other currency of a transfer.
