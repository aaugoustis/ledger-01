import { describe, expect, test } from "bun:test";
import { dateLabel, parseISODate, toISODate } from "./date-picker";

const today = new Date(2026, 9, 29); // Thu, Oct 29, 2026

describe("dateLabel", () => {
  test("names today and yesterday", () => {
    expect(dateLabel(new Date(2026, 9, 29), today)).toBe("Today, Oct 29");
    expect(dateLabel(new Date(2026, 9, 28), today)).toBe("Yesterday, Oct 28");
  });

  test("drops the year only within the current year", () => {
    expect(dateLabel(new Date(2026, 9, 20), today)).toBe("Tue, Oct 20");
    expect(dateLabel(new Date(2025, 11, 1), today)).toBe("Mon, Dec 1, 2025");
  });
});

describe("ISO date helpers", () => {
  test("round-trip without time-zone shifts", () => {
    expect(toISODate(parseISODate("2026-01-31") as Date)).toBe("2026-01-31");
    expect(parseISODate("31/01/2026")).toBeUndefined();
  });
});
