import { describe, expect, test } from "bun:test";
import { BudgetState } from "@ledger-01/api";
import { periodStatus, todayShare } from "./budgets";

describe("budgets", () => {
  test("today marker sits in the middle of today's day", () => {
    expect(todayShare({ state: BudgetState.ACTIVE, daysTotal: 10, daysLeft: 10 })).toBe(0.05);
    expect(todayShare({ state: BudgetState.ACTIVE, daysTotal: 10, daysLeft: 1 })).toBe(0.95);
    expect(
      todayShare({ state: BudgetState.UPCOMING, daysTotal: 10, daysLeft: 10 }),
    ).toBeUndefined();
    expect(todayShare({ state: BudgetState.ENDED, daysTotal: 10, daysLeft: 0 })).toBeUndefined();
  });

  test("status words", () => {
    expect(periodStatus({ state: BudgetState.ACTIVE, daysTotal: 31, daysLeft: 10 })).toBe(
      "10 days left",
    );
    expect(periodStatus({ state: BudgetState.ACTIVE, daysTotal: 31, daysLeft: 1 })).toBe(
      "Last day",
    );
    expect(periodStatus({ state: BudgetState.UPCOMING, daysTotal: 31, daysLeft: 31 })).toBe(
      "Not started",
    );
    expect(periodStatus({ state: BudgetState.ENDED, daysTotal: 31, daysLeft: 0 })).toBe("Ended");
  });
});
