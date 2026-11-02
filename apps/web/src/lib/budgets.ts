import { BudgetRepeat, BudgetState } from "@ledger-01/api";

/** Repeat choices in the order the form offers them. */
export const repeatOptions: { value: BudgetRepeat; label: string }[] = [
  { value: BudgetRepeat.MONTHLY, label: "Every month" },
  { value: BudgetRepeat.WEEKLY, label: "Every week" },
  { value: BudgetRepeat.BIWEEKLY, label: "Every 2 weeks" },
  { value: BudgetRepeat.YEARLY, label: "Every year" },
  { value: BudgetRepeat.DAILY, label: "Every day" },
  { value: BudgetRepeat.ONCE, label: "Once" },
];

const short: Record<BudgetRepeat, string> = {
  [BudgetRepeat.UNSPECIFIED]: "",
  [BudgetRepeat.ONCE]: "Once",
  [BudgetRepeat.DAILY]: "Daily",
  [BudgetRepeat.WEEKLY]: "Weekly",
  [BudgetRepeat.BIWEEKLY]: "Biweekly",
  [BudgetRepeat.MONTHLY]: "Monthly",
  [BudgetRepeat.YEARLY]: "Yearly",
};

/** One-word repeat for HUD serials ("Monthly"). */
export const repeatShort = (r: BudgetRepeat) => short[r];

type ProgressLike = { state: BudgetState; daysTotal: number; daysLeft: number };

/**
 * Where today sits in the period, 0–1, at the middle of today's day: undefined when today is
 * outside the period (no marker).
 */
export function todayShare(p: ProgressLike): number | undefined {
  if (p.state !== BudgetState.ACTIVE || p.daysTotal <= 0) return undefined;
  return (p.daysTotal - p.daysLeft + 0.5) / p.daysTotal;
}

/** Plain-words status for a period, e.g. "10 days left", "Starts in 3 days", "Ended". */
export function periodStatus(p: ProgressLike): string {
  switch (p.state) {
    case BudgetState.ACTIVE:
      return p.daysLeft === 1 ? "Last day" : `${p.daysLeft} days left`;
    case BudgetState.UPCOMING:
      return "Not started";
    default:
      return "Ended";
  }
}
