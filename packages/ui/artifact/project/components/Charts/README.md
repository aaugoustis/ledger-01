# Charts

Recharts wrapped in Module/01 marks: thin lines, 4px rounded bar ends, a dashed `seam` grid, mono axis ticks in `ink-soft`, and a `hud-mark` baseline. Every chart has a hover tooltip: a `shell` card with a `hud-sm` title and tabular values in `ink`.

- `BalanceChart` — one series over time: a 2px `chart-line` stroke over a `signal-tint` area, with a crosshair. A single series has no legend; the panel title names it.
- `FlowChart` — income above the baseline in `chart-income`, expenses below it in `chart-expense`. Direction is a second cue besides color, and a legend sits under the plot. Pass `only` for a single series (no legend).
- `DonutChart` — shares by category color, clockwise from 12 o'clock. The top 7 categories plus "Other" (`cat-slate`, `ellipsis` icon), 2px `shell` gaps between slices, and the total in the center (it shrinks to fit the hole). Every slice of 2% or more gets a callout outside the ring: a 1.5px leader in the slice color, the category's `CategoryIcon` (pass `icon` on each slice; without icons the marker is a color dot), and the share in `hud-sm` `ink` past it. Callouts are pushed apart around the ring so they never overlap, and the leader leaves from inside its own slice. On a narrow panel the shares drop out before the ring gets too small. Labels never sit on slices; still put the ranked list next to it, since it names every slice.

Each takes `money` (`currency`, `minorUnits`) and amounts as BigInt minor units; tooltips format them with the sign and currency.

- Axis ticks use compact numbers (`11.52K`) with enough digits that neighbours never read the same.
- Never use two y-axes. Text never takes a series color.
