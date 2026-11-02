A gadget you'd want on your desk: glossy white plastic parts, signal orange hardware, soft corners with one hard notch, and full HUD markings, like model codes, tick rulers, crosshairs and grill slots. Retro space-age product design meets an engineer's spec sheet. Module/01 is shared by several products: sites (like a personal CV site) and data-dense apps (like Ledger/01, a personal finance app). The rules below hold everywhere; Site surfaces and App surfaces add the patterns for each kind of product.

## Content fundamentals

- Headings are plain words that name the thing: "Projects", "Wallets", "Budgets", "Settings". Don't write them in character ("Mission log", "Operator profile").
- The sci-fi lives in the labels around the copy: section codes (`S-02`), entry codes (`P-01`, `W-03`), serials (`UNIT-01 · REV.2026`), coordinates, build stamps. Keep codes short and consistent: letter, dash, two digits. Each product picks its own letters and uses them the same way on every screen.
- Codes are decoration. Never make a code the only label for something; the plain heading or name always sits next to it.
- Short and concrete. Sites can speak in first person ("I design and build…"); apps speak to the user in second person with plain verbs ("Add wallet", "No transactions in this period"). Bullets start with a verb and carry one point.
- Sentence case for headings and buttons; UPPERCASE only in the mono HUD styles. No emoji.

## Color

- The page is `canvas` (glossy white with a cool cast). Parts are `shell`, sunken areas `shell-recess`, joints `seam`.
- `signal` is generous, like the trim on a product: primary buttons, hinge tabs on cards, HUD code chips, meter fills, grill accents, timeline nodes and feature blocks. Text on `signal` is always `ink`, never white.
- Orange as text uses `signal-ink` (links, active nav, focus ring, emphasized lines); `signal` itself is too light for text on white.
- `studio` (lavender-grey) is the backdrop behind every image, screenshot and portrait, as in a product shoot.
- `visor` is black glass for terminal snippets and dark inset screens, with `shell` text and `visor-mark` microprint.
- `glass` is frosted clear plastic: the nav bar, sticky bars and one or two `GlassPanel`s placed over color.
- `led-on` green marks "available" and "live", always next to a word.
- `hud-mark` is for decorative markings. Microprint that people need to read uses `ink-soft`.
- `danger` fills destructive buttons (with `shell` text) and colors error text; `danger-tint` is the invalid input well.
- `overlay` is the scrim behind dialogs and sheets.
- Domain tokens are shared but only used where they apply: `income-ink`/`expense-ink` for signed money, `cat-red` … `cat-slate` (16 colors) for categories and labels, `chart-*` for chart series. See App surfaces.

## Typography

- Michroma (`--font-display`) for display only: the one hero title on a page, like a name or product name (`display-xl`), section and page headings (`display-lg`), feature block headlines (`display-md`), card and entry titles and the nav mark (`display-sm`). It's wide; keep headings short.
- Manrope (`--font-sans`) for all reading: `body-lg` intros, `body-md` paragraphs and bullets, `body-sm` descriptions and captions, `label` (700) on buttons and nav.
- Space Mono (`--font-mono`) for the HUD: `hud` labels and tags (uppercase, 0.12em), `hud-sm` microprint (uppercase, 0.16em), `hud-num` dates and small figures, `code` on visor.
- Space Mono also sets numbers that matter: `amount-xl` headline figures, `amount-lg` tile and card figures, `amount-md` (700) row figures. Use tabular figures.
- All three load from Google Fonts.

## Shape

- Soft parts with hard notches. Panels and cards are `radius-lg`; hero and feature blocks are `radius-xl`; media frames and inputs are `radius-md`. Each takes one chamfered corner, top-right: `notch-md` on panels and cards, `notch-lg` on hero and feature blocks, `notch-sm` on chips and small tiles.
- Everything you press is a stadium (`radius-full`): buttons, tags, nav bar, grill slots, hinge tabs.
- Hinge tabs are small `signal` stadiums sitting on a card's top edge, like the cube's hinges. Use one per card, and vary the position.

## Depth and material

- Parts float on `shadow-float` (applied as a drop-shadow so the notch keeps it). Buttons are molded with `shadow-mold`; pressed or sunken things use `shadow-press`.
- Glass uses `shadow-glass` plus a 16px backdrop blur. It needs color behind it to read.

## HUD markings

- Every marking must sit on a structure: a code chip on each section label, a tick ruler under each section heading and above the footer, a serial in each panel's top-left corner, a crosshair in its bottom-right corner, two crosshairs on each media frame, and grill slots on hero and feature blocks.
- One crosshair per panel corner pair, never a scatter. Rulers never run through text.
- Sites go loud with markings. Apps keep the same structures but stay calmer where data is dense: lists and forms carry no decoration inside them, only on the panel around them.
- Number in a stable order: sections `S-01`…, entries by a letter for their kind (`P-01` projects, `W-01` wallets), nav links `01`…

## Layout

- Content max `content-max` (1120px), side padding `space-12` (`space-6` on mobile). Sections are `space-24` apart; a section header sits `space-12` above its content.
- The glass NavBar floats over `canvas` and stays sticky. Grids of cards run 2–3 across with `space-8` gaps.

## Imagery and iconography

- Photos and renders go on a `studio` backdrop inside a `radius-md` frame. Use object shots and screenshots as product photos.
- Markings are built from type and simple geometry: plus marks, brackets, ticks, slots, the → arrow in mono.
- Interface icons come from Lucide at a 1.25px stroke, in `ink` or `signal-ink`.
- Products have no logo by default. Set the product's name in Michroma wherever a mark would go, with a short form for tight spaces (`L/01`).

## Site surfaces

Sites (a personal CV site, a project page) are editorial: few pages, big type, loud markings.

- Page order: NavBar → Hero → About (Panel) → Experience (ExperienceEntry stack) → Projects (ProjectCard grid) → ContactBanner → Footer.
- The hero title is your name in `display-xl`, with one portrait on `studio`.
- `signal` carries the timeline nodes and the whole contact block; `signal-ink` sets the role line.
- Numbering: experience `E-01`… (oldest first), projects `P-01`….

## App surfaces

Apps use the same parts for data-dense screens. Pages are a SectionHeader, then Panels.

- Lists live in one Panel with no padding: a mono group header (a day, say) with its total, then rows. Rows hover to `shell-recess`.
- Summary tiles are small Panels (`notch-sm`) with a `hud-sm` label over an `amount-lg` figure.
- Inputs are sunken wells: `shell-recess`, `shadow-press`, `radius-md`, `control-height`. Wrap each in a Field. Dates use DatePicker; choices use Select, Combobox or MultiCombobox.
- Forms flow on their own: dialogs and sheets open on the first field, Select, Combobox and DatePicker move focus to the next field after a pick, and typing on a combobox opens its search (see Combobox). MultiCombobox stays open while toggling and leaves with Tab.
- A rejected input faults like a refusing part: a short stepped servo jitter, one thin `danger` ping leaving its outline, and its label turns `danger` with a blinking fault LED. Never a glow, never longer than a second (see Field).
- Editors open in a sheet: from the right on desktop, from the bottom on phones. A signal stadium FAB opens quick-add on phones.
- Bulk actions sit in a sticky glass stadium bar.
- Serials label panels with the entity's code and one key fact (`W-01 · USD`).

### Money and categories

For products that handle money or categorized records (Ledger/01 today).

- Money as text: `income-ink` for positive amounts and balances, `expense-ink` for negative ones. Both always come with their sign. Every amount is integer minor units under the hood and shows its sign and currency (see Amount). Amounts are entered with MoneyInput; cross-currency transfers use FxConverter.
- `cat-red` … `cat-slate` are the category and label palette: the disc behind a category glyph and the dot beside a label. Decorative fills; glyphs on them are `ink`. Category glyphs are Lucide icons on a palette disc (CategoryIcon).
- Reports: a row of summary tiles, then chart Panels (balance, cash flow, category donuts with ranked lists). Series colors come from `chart-line`, `chart-income`, `chart-expense` and the category palette (see Charts).
- Budgets: a grid of budget Panels (name, spent of budgeted, BudgetMeter, "left" or "over by", per-day allowance, period). The detail adds the meter with a today caption, category and wallet donuts, a history of periods with small meters, and the period's expenses.

## Don'ts

- Don't put white text on `signal`, and don't use `signal` as small text on white.
- Don't show an amount without its sign, and don't rely on `income-ink`/`expense-ink` alone.
- Don't put more than one notch on a part, and don't notch a stadium.
- Don't use gradients, neon glows or dark mode. This system is lit like a white studio.
- Don't write headings in character, and don't let a HUD code stand in for a heading. Keep the copy plain.
