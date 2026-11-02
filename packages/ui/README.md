# @ledger-01/ui

The component kit for the Ledger/01 web app: [shadcn/ui](https://ui.shadcn.com) components themed
with the **Module/01** design system (glossy white parts, signal orange trim, HUD markings).

## Layout

- `src/components` — the components the app imports, as
  `@ledger-01/ui/components/<name>`.
- `src/lib`, `src/hooks` — helpers shared by the components and the app (money formatting, form
  flow, fault feedback).
- `src/styles` — global CSS and the Tailwind theme.
- `tokens/tokens.json` — the design tokens. Run `bun run --cwd packages/ui tokens` after editing it,
  to regenerate the CSS variables. App code styles only through these tokens, never raw hex values.
- `artifact/` — the design-system documentation, not used by the app:
  - `artifact/project/README.md` — the design guidelines (content, color, type, surfaces);
  - `artifact/project/components/<Name>/` — a guideline and preview per component;
  - `artifact/base/` — the kit built as one classic script (`task ds:bundle`), so the previews can
    render the real components.

## Adding a component

Add it under `src/components`, style it with tokens, and document it in `artifact/project/components/`
in the same change, then run `task ds:bundle`.
