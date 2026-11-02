# CategoryIcon

A category's Lucide glyph in `ink` on a disc of its palette color: the marker beside every transaction, budget and category row.

Provide `icon` (a Lucide name from the curated set, e.g. `shopping-cart`) and `color` (a `cat-*` palette token). Optional `size` (`sm` 28px, `md` 36px default, `lg` 48px) and `label` (accessible name; omit when the category name sits next to it). `ColorDot` is the 10px dot used for labels.

- Glyphs are 1.25px Lucide strokes in `ink`; every `cat-*` color keeps ink at 5.8:1 or better.
- The disc is molded: a soft top highlight and bottom shade, no outline.
- Transfers use `arrow-left-right` on `cat-slate`.
- Pair it with the category name; color alone never identifies a category.
