import { describe, expect, test } from "bun:test";
import { currentMonth, parsePeriod, shiftPeriod } from "./period";

describe("shiftPeriod", () => {
  test("steps whole months", () => {
    expect(shiftPeriod({ from: "2026-10-01", to: "2026-10-31" }, -1)).toEqual({
      from: "2026-09-01",
      to: "2026-09-30",
    });
    expect(shiftPeriod({ from: "2026-12-01", to: "2026-12-31" }, 1)).toEqual({
      from: "2027-01-01",
      to: "2027-01-31",
    });
    expect(shiftPeriod({ from: "2026-01-01", to: "2026-03-31" }, 1)).toEqual({
      from: "2026-04-01",
      to: "2026-06-30",
    });
  });

  test("steps arbitrary ranges by their length", () => {
    expect(shiftPeriod({ from: "2026-10-05", to: "2026-10-11" }, 1)).toEqual({
      from: "2026-10-12",
      to: "2026-10-18",
    });
  });
});

describe("parsePeriod", () => {
  test("falls back to the current month", () => {
    expect(parsePeriod(new URLSearchParams("from=bad"))).toEqual(currentMonth());
    expect(parsePeriod(new URLSearchParams("from=2026-10-31&to=2026-10-01"))).toEqual(
      currentMonth(),
    );
    expect(parsePeriod(new URLSearchParams("from=2026-10-01&to=2026-10-31"))).toEqual({
      from: "2026-10-01",
      to: "2026-10-31",
    });
  });
});
