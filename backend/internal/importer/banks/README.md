# Banks

The statement import (`internal/importer`) understands universal ideas: a **purchase**, a
**discount** on it (always smaller), a **refund** of it (in full or in part), a **double charge**
and its refund, and duplicates. It knows no bank's wording. This folder holds that wording, one
file per bank, so discounts and refunds get matched to their purchase:

| Bank | Country | File |
|---|---|---|
| Itaú | Uruguay | [`uy/itau.go`](uy/itau.go) |

Any other bank imports as "Other bank". Fingerprints, duplicate detection and the column mapping
still work, but discount and refund rows stay as their own rows.

## Add your bank

PRs that add a bank are welcome. They're small, self-contained, and they help everyone who banks
there.

1. **Read a statement.** Export one and note how your bank words:
   - purchases (e.g. `COMPRA <merchant>`, `POS <merchant>`; leave this out if purchases have no
     prefix);
   - discounts (e.g. Uruguay's VAT refund `REDIVA 19210<merchant>`);
   - refunds and cancellations (e.g. `DEVOLUCION <merchant>`).
2. **Add `banks/<country>/<bank>.go`** (an ISO 3166-1 alpha-2 country code, lower case, as the
   folder and package name). Use [`uy/itau.go`](uy/itau.go) as the model:

   ```go
   var Northbank = importer.Bank{
       ID:      "zz-northbank", // "<country>-<bank>"
       Name:    "Northbank",
       Country: "ZZ",
       Patterns: []importer.Pattern{
           importer.NewPattern(importer.RolePurchase, `^\s*card\s+(?P<merchant>.+?)\s*$`, 0),
           importer.NewPattern(importer.RoleDiscount, `^\s*cashback\s+(?P<merchant>.+?)\s*$`, 0),
           importer.NewPattern(importer.RoleRefund, `^\s*refund\s+(?P<merchant>.+?)\s*$`, 0),
       },
   }
   ```

   - Patterns are case-insensitive and need a `merchant` group. The merchant is how a discount or
     refund finds its purchase; it may be truncated differently in each row.
   - The last argument is how long after the purchase the row may come (0 = 7 days for discounts,
     30 for refunds).
   - The first pattern that matches wins.
3. **Register it** in `All()` in [`banks.go`](banks.go).
4. **Test it** in `<bank>_test.go` next to it:
   - `Classify` cases for each pattern;
   - a `DetectBank` case;
   - optionally a pairing case on a fixture CSV in `internal/importer/testdata/`.

   **Use made-up data only**: no real names, account numbers, merchants you can be traced to,
   or amounts from your statements.
5. **Check and open the PR:** run `task test` and `task lint`, then open a PR that names the bank
   and country and shows one anonymised sample line per pattern.
