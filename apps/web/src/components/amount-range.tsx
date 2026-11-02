"use client";

import { Field } from "@ledger-01/ui/components/field";
import { MoneyInput } from "@ledger-01/ui/components/money-input";
import { useId } from "react";

/** Min and max amount fields for filters, in one currency. Empty means no bound. */
export function AmountRange({
  currency,
  min,
  max,
  onChange,
}: {
  currency: string;
  min: string;
  max: string;
  onChange: (range: { min: string; max: string }) => void;
}) {
  const id = useId();
  return (
    <div className="grid grid-cols-2 gap-3">
      <Field label="Min amount" htmlFor={`${id}-min`}>
        <MoneyInput
          id={`${id}-min`}
          currency={currency}
          value={min}
          onChange={(e) => onChange({ min: e.target.value, max })}
        />
      </Field>
      <Field label="Max amount" htmlFor={`${id}-max`}>
        <MoneyInput
          id={`${id}-max`}
          currency={currency}
          value={max}
          onChange={(e) => onChange({ min, max: e.target.value })}
        />
      </Field>
    </div>
  );
}
