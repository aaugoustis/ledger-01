import { cn } from "@ledger-01/ui/lib/utils";

/**
 * Budget progress: a sunken track filled to the share spent (signal while within budget,
 * danger once over), with a thin "today" tick at the share of the period that has passed.
 * Pair it with the figures in text; the fill color is never the only cue.
 */
export function BudgetMeter({
  spent,
  budget,
  elapsed,
  label,
  size = "md",
  className,
}: {
  /** Minor units spent so far. */
  spent: bigint;
  /** Budgeted minor units (> 0). */
  budget: bigint;
  /** Share of the period that has passed, 0–1; omit to hide the today tick. */
  elapsed?: number;
  /** Accessible name, e.g. "Food budget". */
  label: string;
  size?: "sm" | "md";
  className?: string;
}) {
  const share = budget > 0n ? Number((spent * 10000n) / budget) / 10000 : 0;
  const over = spent > budget;
  const fill = Math.min(Math.max(share, 0), 1);
  return (
    // biome-ignore lint/a11y/useSemanticElements: a native <meter> can't hold the today tick or this track
    <div
      role="meter"
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={Math.round(share * 100)}
      aria-valuetext={`${Math.round(share * 100)}% spent`}
      data-over={over || undefined}
      className={cn("relative", size === "md" ? "py-1.5" : "py-1", className)}
    >
      <div
        className={cn(
          "overflow-hidden rounded-full bg-shell-recess shadow-press",
          size === "md" ? "h-3" : "h-2",
        )}
      >
        <div
          className={cn("h-full rounded-full", over ? "bg-danger" : "bg-signal")}
          style={{ width: `${fill * 100}%` }}
        />
      </div>
      {elapsed !== undefined && elapsed > 0 && elapsed < 1 && (
        <div
          aria-hidden
          className="absolute inset-y-0 w-px bg-ink"
          style={{ left: `${elapsed * 100}%` }}
        >
          <span className="absolute -top-0.5 left-1/2 size-1.5 -translate-x-1/2 rotate-45 bg-ink" />
        </div>
      )}
    </div>
  );
}
