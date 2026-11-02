"use client";

import { type Transaction, TransactionKind, type Wallet } from "@ledger-01/api";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@ledger-01/ui/components/sheet";
import { TransactionForm } from "@/app/(app)/wallets/[id]/transaction-form";
import { useCategories, useCurrencies, useLabels, useWallets } from "@/lib/api/hooks";
import { useMediaQuery } from "@/lib/use-media-query";

/**
 * The new/edit transaction form in a sheet: on the right on desktop, a bottom sheet capped below
 * the top on phones.
 */
export function TransactionSheet({
  open,
  onOpenChange,
  wallet,
  editing,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  wallet: Wallet | undefined;
  editing: Transaction | null;
}) {
  const desktop = useMediaQuery("(min-width: 768px)");
  const { wallets } = useWallets();
  const { categories } = useCategories();
  const { labels } = useLabels();
  const { minorUnits } = useCurrencies();

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side={desktop ? "right" : "bottom"}
        className={desktop ? "overflow-y-auto" : "max-h-[92dvh] overflow-y-auto"}
      >
        <SheetHeader>
          <SheetTitle>{editing ? editTitle(editing) : "New transaction"}</SheetTitle>
          <SheetDescription>{wallet?.name}</SheetDescription>
        </SheetHeader>
        {wallet && open && (
          <div className="px-4 pb-6">
            <TransactionForm
              key={editing?.id ?? "new"}
              wallet={wallet}
              wallets={wallets}
              categories={categories}
              labels={labels}
              minorUnits={minorUnits}
              editing={editing}
              onDone={() => onOpenChange(false)}
            />
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}

function editTitle(t: Transaction) {
  if (t.kind === TransactionKind.TRANSFER_IN || t.kind === TransactionKind.TRANSFER_OUT)
    return "Edit transfer";
  return "Edit transaction";
}
