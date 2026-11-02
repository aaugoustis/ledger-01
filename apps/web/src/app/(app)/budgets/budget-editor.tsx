"use client";

import type { Budget } from "@ledger-01/api";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@ledger-01/ui/components/sheet";
import { useCategories, useCurrencies, useWallets } from "@/lib/api/hooks";
import { useMediaQuery } from "@/lib/use-media-query";
import { BudgetForm } from "./budget-form";

/** The budget form in a sheet: from the right on desktop, from the bottom on phones. */
export function BudgetEditor({
  open,
  onOpenChange,
  editing,
  onSaved,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  editing: Budget | null;
  onSaved?: (budget: Budget) => void;
}) {
  const desktop = useMediaQuery("(min-width: 768px)");
  const { wallets } = useWallets();
  const { categories } = useCategories();
  const { minorUnits } = useCurrencies();
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side={desktop ? "right" : "bottom"}
        className={desktop ? "overflow-y-auto" : "max-h-[92dvh] overflow-y-auto"}
      >
        <SheetHeader>
          <SheetTitle>{editing ? "Edit budget" : "New budget"}</SheetTitle>
          <SheetDescription>
            A spending limit that starts over every period. Only expenses count.
          </SheetDescription>
        </SheetHeader>
        {open && (
          <div className="px-4 pb-6">
            <BudgetForm
              key={editing?.id ?? "new"}
              editing={editing}
              wallets={wallets}
              categories={categories}
              minorUnits={minorUnits}
              onDone={(b) => {
                onOpenChange(false);
                onSaved?.(b);
              }}
            />
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}
