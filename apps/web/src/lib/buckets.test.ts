import { describe, expect, test } from "bun:test";
import { bucketLabels, defaultBucket, parseBucket, periodForBucket } from "./buckets";

describe("buckets", () => {
  test("defaults by period length", () => {
    expect(defaultBucket({ from: "2026-10-01", to: "2026-10-31" })).toBe("day");
    expect(defaultBucket({ from: "2026-01-01", to: "2026-06-30" })).toBe("week");
    expect(defaultBucket({ from: "2024-01-01", to: "2026-12-31" })).toBe("month");
    expect(defaultBucket({ from: "2025-11-01", to: "2026-10-31" })).toBe("month");
  });

  test("each grouping gets a span that ends with the anchor's month", () => {
    expect(periodForBucket("day", "2026-10-31")).toEqual({ from: "2026-10-01", to: "2026-10-31" });
    expect(periodForBucket("week", "2026-10-15")).toEqual({ from: "2026-08-01", to: "2026-10-31" });
    expect(periodForBucket("month", "2026-02-10")).toEqual({
      from: "2025-03-01",
      to: "2026-02-28",
    });
  });

  test("parses or falls back", () => {
    expect(parseBucket("week", { from: "2026-10-01", to: "2026-10-31" })).toBe("week");
    expect(parseBucket("bogus", { from: "2026-10-01", to: "2026-10-31" })).toBe("day");
  });

  test("labels", () => {
    expect(bucketLabels("2026-10-05", "day")).toEqual({
      label: "Oct 5",
      tooltipLabel: "Mon, Oct 5, 2026",
    });
    expect(bucketLabels("2026-10-05", "week").tooltipLabel).toBe("Week of Oct 5, 2026");
    expect(bucketLabels("2026-10-01", "month")).toEqual({
      label: "Oct",
      tooltipLabel: "October 2026",
    });
  });
});
