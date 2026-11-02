# MultiCombobox

A searchable multi-choice picker for "which ones count" fields, like a budget's wallets and categories. Selecting nothing means all, and the trigger then reads `allLabel` ("All wallets").

Takes the same option props as Combobox (`options` or `groups`, `getKey`, `getSearchText`, `renderOption`), plus `value` (selected keys), `allLabel` and `noun` (plural, for "3 wallets").

- The list stays open while you toggle options, with a check on each selected one. Escape or a click outside closes it back onto the trigger, and Tab moves on to the next field.
- Typing on the focused trigger opens it with that text as the search, like Combobox.
- With one option selected, the trigger shows it; with more, a count.
- "Use all …" under the list clears the selection.
- Use it in forms; for report filters in a toolbar, a smaller secondary button works the same way.
