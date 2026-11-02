import { formatMoney, type SignDisplay } from "@ledger-01/ui/lib/money";
import { cn } from "@ledger-01/ui/lib/utils";

const sizes = {
  sm: "text-hud-num",
  md: "text-amount-md",
  lg: "text-amount-lg",
  xl: "text-amount-xl",
};

/**
 * A money figure in tabular mono. Positive values use income-ink, negative expense-ink, zero ink.
 * The sign is always printed so color is never the only cue.
 */
export function Amount({
  minor,
  currency,
  minorUnits,
  sign = "always",
  code = true,
  size = "md",
  tone = "auto",
  className,
}: {
  minor: bigint;
  currency: string;
  minorUnits: number;
  sign?: SignDisplay;
  code?: boolean;
  size?: keyof typeof sizes;
  /** "auto" colors by sign; "plain" keeps ink; "muted" uses ink-soft. */
  tone?: "auto" | "plain" | "muted";
  className?: string;
}) {
  const color =
    tone === "plain"
      ? "text-ink"
      : tone === "muted"
        ? "text-ink-soft"
        : minor > 0n
          ? "text-income-ink"
          : minor < 0n
            ? "text-expense-ink"
            : "text-ink";
  return (
    <span className={cn("tabular whitespace-nowrap", sizes[size], color, className)}>
      {formatMoney(minor, currency, minorUnits, { sign, code })}
    </span>
  );
}
