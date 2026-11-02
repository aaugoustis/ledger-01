import { RecurringFrequency } from "@ledger-01/api";

/** Repeat choices in the order the form offers them ("never" first). */
export const frequencyOptions: { value: RecurringFrequency; label: string }[] = [
  { value: RecurringFrequency.UNSPECIFIED, label: "Never" },
  { value: RecurringFrequency.DAILY, label: "Every day" },
  { value: RecurringFrequency.EVERY_2_DAYS, label: "Every 2 days" },
  { value: RecurringFrequency.WEEKDAYS, label: "Every weekday" },
  { value: RecurringFrequency.WEEKLY, label: "Every week" },
  { value: RecurringFrequency.EVERY_2_WEEKS, label: "Every 2 weeks" },
  { value: RecurringFrequency.EVERY_4_WEEKS, label: "Every 4 weeks" },
  { value: RecurringFrequency.MONTHLY, label: "Every month" },
  { value: RecurringFrequency.EVERY_2_MONTHS, label: "Every 2 months" },
  { value: RecurringFrequency.EVERY_3_MONTHS, label: "Every 3 months" },
  { value: RecurringFrequency.EVERY_6_MONTHS, label: "Every 6 months" },
  { value: RecurringFrequency.YEARLY, label: "Every year" },
];

const labels = new Map(frequencyOptions.map((o) => [o.value, o.label]));

/** "Every month", or "" for a transaction that doesn't repeat. */
export function frequencyLabel(f: RecurringFrequency | undefined): string {
  return f ? (labels.get(f) ?? "") : "";
}

export type RepeatEnd = "never" | "date" | "count";

export const endOptions: { value: RepeatEnd; label: string }[] = [
  { value: "never", label: "Never" },
  { value: "date", label: "On a date" },
  { value: "count", label: "After a number of times" },
];

/** The form's repeat fields. */
export type RepeatState = {
  frequency: RecurringFrequency;
  end: RepeatEnd;
  endDate: string;
  count: string;
};

/** Form state for a row's rule, or "never" for a row that doesn't repeat. */
export function repeatState(r?: { frequency: RecurringFrequency; endDate: string }): RepeatState {
  return {
    frequency: r?.frequency ?? RecurringFrequency.UNSPECIFIED,
    end: r?.endDate ? "date" : "never",
    endDate: r?.endDate ?? "",
    count: "",
  };
}

export type RepeatInput = { frequency: RecurringFrequency; endDate: string; endCount: number };

/**
 * The API's repeat for the form state, or an error message for the field at fault. A count must
 * be a whole number from 1 to 1000, and an end date can't come before the first date.
 */
export function repeatInput(
  s: RepeatState,
  firstDate: string,
): { ok: true; value: RepeatInput } | { ok: false; field: "endDate" | "count"; message: string } {
  const value: RepeatInput = { frequency: s.frequency, endDate: "", endCount: 0 };
  if (s.frequency === RecurringFrequency.UNSPECIFIED || s.end === "never")
    return { ok: true, value };
  if (s.end === "date") {
    if (!s.endDate) return { ok: false, field: "endDate", message: "Choose the last date" };
    if (s.endDate < firstDate)
      return { ok: false, field: "endDate", message: "Choose a date on or after the first one" };
    return { ok: true, value: { ...value, endDate: s.endDate } };
  }
  const n = Number(s.count.trim());
  if (!/^\d+$/.test(s.count.trim()) || n < 1 || n > 1000)
    return { ok: false, field: "count", message: "Enter a number from 1 to 1000" };
  return { ok: true, value: { ...value, endCount: n } };
}

/** True when two schedules differ, comparing what the form would send. */
export function repeatChanged(a: RepeatInput, b: RepeatInput): boolean {
  return a.frequency !== b.frequency || a.endDate !== b.endDate || a.endCount !== b.endCount;
}
