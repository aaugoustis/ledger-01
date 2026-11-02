// Period helpers for the "oct. 01, 2026 – oct. 31, 2026" style range with prev/next arrows.
// Dates are "YYYY-MM-DD" strings; arithmetic is done in UTC so time zones never shift a day.

import {
  addDays,
  addMonths,
  differenceInCalendarDays,
  endOfMonth,
  format,
  isSameDay,
  startOfMonth,
} from "date-fns";

export type Period = { from: string; to: string };

const toDate = (s: string) => {
  const [y = 1970, m = 1, d = 1] = s.split("-").map(Number);
  return new Date(y, m - 1, d);
};
export const toISODate = (d: Date) => format(d, "yyyy-MM-dd");

export function today(): string {
  return toISODate(new Date());
}

export function currentMonth(now = new Date()): Period {
  return { from: toISODate(startOfMonth(now)), to: toISODate(endOfMonth(now)) };
}

/** True when the period is exactly whole calendar months. */
function wholeMonths(p: Period): number | null {
  const from = toDate(p.from);
  const to = toDate(p.to);
  if (from.getDate() !== 1 || !isSameDay(to, endOfMonth(to))) return null;
  return (to.getFullYear() - from.getFullYear()) * 12 + to.getMonth() - from.getMonth() + 1;
}

/** Moves the period one span back (-1) or forward (+1). Month ranges step by months. */
export function shiftPeriod(p: Period, dir: 1 | -1): Period {
  const months = wholeMonths(p);
  if (months) {
    const from = addMonths(toDate(p.from), dir * months);
    return { from: toISODate(from), to: toISODate(endOfMonth(addMonths(from, months - 1))) };
  }
  const days = differenceInCalendarDays(toDate(p.to), toDate(p.from)) + 1;
  return {
    from: toISODate(addDays(toDate(p.from), dir * days)),
    to: toISODate(addDays(toDate(p.to), dir * days)),
  };
}

export function formatPeriod(p: Period): string {
  return `${format(toDate(p.from), "MMM d, yyyy")} – ${format(toDate(p.to), "MMM d, yyyy")}`;
}

export function formatDay(s: string): string {
  return format(toDate(s), "EEE, MMM d, yyyy");
}

export function parsePeriod(params: URLSearchParams): Period {
  const from = params.get("from");
  const to = params.get("to");
  const re = /^\d{4}-\d{2}-\d{2}$/;
  if (from && to && re.test(from) && re.test(to) && from <= to) return { from, to };
  return currentMonth();
}
