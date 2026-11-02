"use client";

import { cn } from "@ledger-01/ui/lib/utils";
import type * as React from "react";

/**
 * Decimal amount input with the currency code as a suffix. The value is the raw string the user
 * typed; parse it with `parseAmount` from lib/money. On phones the figure is amount-lg: inputs under
 * 16px make mobile browsers zoom in on focus.
 */
export function MoneyInput({
  currency,
  className,
  ...props
}: Omit<React.ComponentProps<"input">, "type"> & { currency: string }) {
  return (
    <div
      data-slot="money-input"
      data-fault-part
      className={cn(
        "flex h-control-height w-full items-center rounded-md border border-transparent bg-shell-recess shadow-press transition-[box-shadow,border-color] focus-within:border-signal-ink focus-within:ring-[3px] focus-within:ring-ring/30 has-[[aria-invalid=true]]:border-danger has-[[aria-invalid=true]]:bg-danger-tint",
        className,
      )}
    >
      <input
        type="text"
        inputMode="decimal"
        autoComplete="off"
        placeholder="0.00"
        className="tabular h-full min-w-0 flex-1 bg-transparent pl-4 text-right text-amount-md text-ink outline-none max-sm:text-amount-lg placeholder:text-ink-soft disabled:opacity-50"
        {...props}
      />
      <span className="hud px-3 text-ink-soft">{currency}</span>
    </div>
  );
}
