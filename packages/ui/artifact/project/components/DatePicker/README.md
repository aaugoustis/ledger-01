# DatePicker

A calendar date in a sunken well that opens a Module/01 month calendar, with Today and Yesterday shortcuts. `Calendar` is the month grid on its own.

Provide `value` and `onChange` as `"YYYY-MM-DD"` strings (calendar dates have no time zone), plus `id` for its Field. Optional `placeholder` and `disabled`. `Calendar` takes react-day-picker props (`mode="single"`, `selected`, `onSelect`, `month`, `onMonthChange`).

- The field matches TextInput: `shell-recess` well, `shadow-press`, `radius-md`, `control-height`, a 1.25px `calendar-days` glyph in `ink-soft`. Open, it takes a `signal-ink` border.
- The label stays short: "Today, Oct 29", "Yesterday, Oct 28", "Tue, Oct 20"; the year appears only outside the current year.
- The calendar sits in a `shell` popover with `shadow-float` and `radius-lg`. The caption is `hud` ("OCTOBER 2026"), weekdays are `hud-sm` in `ink-soft`, and weeks start on Monday.
- Days are 36px stadiums in tabular mono. Selected: `signal` fill with `ink` text and `shadow-mold`. Today: a `signal-ink` ring. Other months' days: `hud-mark`.
- Month arrows are small molded `shell` stadiums at the caption's right.
