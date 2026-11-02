"use client";

import { editFx, type FxCurrencies, type FxField, type FxState } from "@ledger-01/ui/lib/fx";
import { cn } from "@ledger-01/ui/lib/utils";
import { useId } from "react";
import { FieldLabel } from "./field";
import { MoneyInput } from "./money-input";

type Currency = { code: string; minorUnits: number };

/**
 * Three linked inputs for a cross-currency transfer: amount sent, amount received and rate.
 * Editing one recomputes the least recently edited of the other two (see lib/fx.ts).
 * Controlled: keep the FxState in the parent (start from `initialFxState()`).
 */
export function FxConverter({
  id: rootId,
  from,
  to,
  value,
  onChange,
  invalid,
  className,
}: {
  /** Id of the root, e.g. to fault all three fields with `fault(id)`. */
  id?: string;
  from: Currency;
  to: Currency;
  value: FxState;
  onChange: (next: FxState) => void;
  invalid?: boolean;
  className?: string;
}) {
  const id = useId();
  const c: FxCurrencies = { fromMinorUnits: from.minorUnits, toMinorUnits: to.minorUnits };
  const edit = (field: FxField) => (e: React.ChangeEvent<HTMLInputElement>) =>
    onChange(editFx(value, field, e.target.value, c));
  // The field that will be recomputed next is the oldest one; mark it so users can see it.
  const next = value.order[0];

  // Columns follow the converter's own width (it lives in sheets and narrow cards), not the viewport.
  return (
    <div id={rootId} data-fault-group className={cn("@container", className)}>
      <div className="grid gap-3 @lg:grid-cols-3">
        <FxRow
          id={`${id}-from`}
          label={`Sent (${from.code})`}
          computed={next === "from"}
          invalid={invalid}
        >
          <MoneyInput
            id={`${id}-from`}
            currency={from.code}
            value={value.from}
            onChange={edit("from")}
            aria-invalid={invalid || undefined}
          />
        </FxRow>
        <FxRow
          id={`${id}-to`}
          label={`Received (${to.code})`}
          computed={next === "to"}
          invalid={invalid}
        >
          <MoneyInput
            id={`${id}-to`}
            currency={to.code}
            value={value.to}
            onChange={edit("to")}
            aria-invalid={invalid || undefined}
          />
        </FxRow>
        <FxRow
          id={`${id}-rate`}
          label={`Rate (${to.code} per ${from.code})`}
          computed={next === "rate"}
          invalid={invalid}
        >
          <MoneyInput
            id={`${id}-rate`}
            currency="×"
            placeholder="0.000000"
            value={value.rate}
            onChange={edit("rate")}
            aria-invalid={invalid || undefined}
          />
        </FxRow>
      </div>
    </div>
  );
}

function FxRow({
  id,
  label,
  computed,
  invalid,
  children,
}: {
  id: string;
  label: string;
  computed: boolean;
  invalid?: boolean;
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-w-0 flex-col gap-1.5">
      <div className="flex items-baseline justify-between gap-2">
        <FieldLabel htmlFor={id} invalid={invalid}>
          {label}
        </FieldLabel>
        {computed && <span className="hud-sm text-hud-mark">auto</span>}
      </div>
      {children}
    </div>
  );
}
