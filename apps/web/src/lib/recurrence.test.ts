import { describe, expect, test } from "bun:test";
import { RecurringFrequency } from "@ledger-01/api";
import {
  frequencyLabel,
  frequencyOptions,
  repeatChanged,
  repeatInput,
  repeatState,
} from "./recurrence";

describe("frequencyOptions", () => {
  test("offers never first, then every frequency once", () => {
    expect(frequencyOptions[0]?.value).toBe(RecurringFrequency.UNSPECIFIED);
    const values = frequencyOptions.map((o) => o.value);
    expect(new Set(values).size).toBe(values.length);
    expect(values.length).toBe(12);
  });

  test("labels", () => {
    expect(frequencyLabel(RecurringFrequency.EVERY_2_WEEKS)).toBe("Every 2 weeks");
    expect(frequencyLabel(RecurringFrequency.UNSPECIFIED)).toBe("");
    expect(frequencyLabel(undefined)).toBe("");
  });
});

describe("repeatState", () => {
  test("a row that doesn't repeat", () => {
    expect(repeatState()).toEqual({
      frequency: RecurringFrequency.UNSPECIFIED,
      end: "never",
      endDate: "",
      count: "",
    });
  });

  test("a rule with an end date reads back as 'on a date'", () => {
    const s = repeatState({ frequency: RecurringFrequency.MONTHLY, endDate: "2027-03-31" });
    expect(s.end).toBe("date");
    expect(s.endDate).toBe("2027-03-31");
  });
});

describe("repeatInput", () => {
  const base = repeatState({ frequency: RecurringFrequency.WEEKLY, endDate: "" });

  test("never sends no frequency", () => {
    expect(repeatInput(repeatState(), "2026-10-30")).toEqual({
      ok: true,
      value: { frequency: RecurringFrequency.UNSPECIFIED, endDate: "", endCount: 0 },
    });
  });

  test("ignores end fields when the end is never", () => {
    const r = repeatInput({ ...base, endDate: "2026-01-01", count: "x" }, "2026-10-30");
    expect(r).toEqual({
      ok: true,
      value: { frequency: RecurringFrequency.WEEKLY, endDate: "", endCount: 0 },
    });
  });

  test("an end date must be set and not before the first date", () => {
    expect(repeatInput({ ...base, end: "date" }, "2026-10-30")).toMatchObject({
      ok: false,
      field: "endDate",
    });
    expect(
      repeatInput({ ...base, end: "date", endDate: "2026-10-29" }, "2026-10-30"),
    ).toMatchObject({ ok: false, field: "endDate" });
    expect(repeatInput({ ...base, end: "date", endDate: "2026-10-30" }, "2026-10-30")).toEqual({
      ok: true,
      value: { frequency: RecurringFrequency.WEEKLY, endDate: "2026-10-30", endCount: 0 },
    });
  });

  test("a count is a whole number from 1 to 1000", () => {
    for (const bad of ["", "0", "1001", "2.5", "-3", "ten"]) {
      expect(repeatInput({ ...base, end: "count", count: bad }, "2026-10-30")).toMatchObject({
        ok: false,
        field: "count",
      });
    }
    expect(repeatInput({ ...base, end: "count", count: " 12 " }, "2026-10-30")).toEqual({
      ok: true,
      value: { frequency: RecurringFrequency.WEEKLY, endDate: "", endCount: 12 },
    });
  });
});

describe("repeatChanged", () => {
  const monthly = { frequency: RecurringFrequency.MONTHLY, endDate: "", endCount: 0 };
  test("compares what would be sent", () => {
    expect(repeatChanged(monthly, { ...monthly })).toBe(false);
    expect(repeatChanged(monthly, { ...monthly, endDate: "2027-01-01" })).toBe(true);
    expect(repeatChanged(monthly, { ...monthly, frequency: RecurringFrequency.YEARLY })).toBe(true);
    expect(repeatChanged(monthly, { ...monthly, endCount: 3 })).toBe(true);
  });
});
