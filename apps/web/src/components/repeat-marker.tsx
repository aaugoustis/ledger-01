import type { Transaction } from "@ledger-01/api";
import { Repeat } from "lucide-react";
import { frequencyLabel } from "@/lib/recurrence";

/** A small repeat icon for occurrences of a recurring transaction, with its schedule as text. */
export function RepeatMarker({ transaction }: { transaction: Transaction }) {
  if (!transaction.recurringRuleId) return null;
  const label = frequencyLabel(transaction.recurrence?.frequency) || "Repeats";
  return (
    <span className="inline-flex shrink-0 items-center text-ink-soft" title={label}>
      <Repeat strokeWidth={1.25} className="size-3.5" aria-hidden />
      <span className="sr-only">{`Repeats: ${label}`}</span>
    </span>
  );
}
