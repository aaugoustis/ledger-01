# Combobox

A searchable single-choice picker: a molded `shell` stadium trigger opening a `shell` popover with a search well and a list. Use it for categories, currencies and any list too long for a Select.

Provide `value` and `onChange` (option keys), `options` (or `groups` with headings), `getKey`, and `renderOption`. Optional `getSearchText`, `renderValue`, `placeholder`, `searchPlaceholder`, `emptyText`, `footer` (e.g. "Edit categories →" in `hud-sm` `signal-ink`), `id`, `invalid`.

- The trigger shows the selected option (glyph + name) or the placeholder in `ink-soft`, with a `chevrons-up-down` glyph at 1.25px.
- Options are rows with the option's own glyph; the selected one carries a check at the end.

## Form flow

Forms built from kit controls move focus the way people fill them in, with no per-form wiring:

- `Dialog` and `Sheet` open with focus on their first field (not the close button or tabs). If that field is an empty Combobox, its list opens too, once the entrance animation has finished.
- **Type to search**: a letter on the focused trigger opens the list with that letter as the search.
- **Pick to advance**: choosing an option (click or Enter) closes the list and focuses the next field. `Select` and `DatePicker` do the same. Escape returns to the trigger.
- Tab follows layout order, so lay fields out in the order they are filled.
- Rejected fields use the fault feedback (see Field).
