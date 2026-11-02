# MoneyInput

A sunken decimal well for amounts, right-aligned in tabular mono, with the currency code as a suffix.

Provide `currency` and the usual input props (`value`, `onChange`, `id`). The value is the raw string the person typed; parse it with `money.parseAmount(value, minorUnits)`, which returns integer minor units or null.

- `shell-recess` fill with `shadow-press`, `radius-md`, `control-height`. Focus draws a `signal-ink` border and a soft ring.
- Opens the decimal keypad on phones (`inputMode="decimal"`). On phones the figure grows to `amount-lg`: inputs under 16px make mobile browsers zoom in on focus. Search fields in pickers use `body-md` on phones for the same reason.
- Invalid (`aria-invalid`): `danger` border on a `danger-tint` well; put the reason under it in a Field.
- Amounts are entered positive; the transaction kind gives the direction.
