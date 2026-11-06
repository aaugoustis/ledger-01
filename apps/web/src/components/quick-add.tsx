"use client";

import { Button } from "@ledger-01/ui/components/button";
import { Plus } from "lucide-react";
import { usePathname, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { TransactionSheet } from "@/components/transaction-sheet";
import { useWallets } from "@/lib/api/hooks";
import { lastWalletId, startingWallet } from "@/lib/last-wallet";

/**
 * Adds a transaction from outside a wallet page: a New transaction button on desktop, the pinned
 * quick-add button on phones, and the installed app's New transaction shortcut, which lands on the
 * page with ?new=transaction. It starts in the wallet last added to; the form can pick another.
 */
export function QuickAdd() {
  const pathname = usePathname();
  const params = useSearchParams();
  const { wallets, byId, isPending } = useWallets();
  const [open, setOpen] = useState(false);
  const [walletId, setWalletId] = useState("");
  const canAdd = Boolean(startingWallet(wallets, null));

  function start() {
    const w = startingWallet(wallets, lastWalletId());
    if (!w) return;
    setWalletId(w.id);
    setOpen(true);
  }

  // Opened by the app shortcut: wait for the wallets, then drop the parameter so a reload or the
  // back button doesn't open the form again. The native replaceState syncs with the Next router
  // without fetching the page again.
  const requested = params.get("new") === "transaction";
  useEffect(() => {
    if (!requested || isPending) return;
    const w = startingWallet(wallets, lastWalletId());
    if (w) {
      setWalletId(w.id);
      setOpen(true);
    }
    const rest = new URLSearchParams(params);
    rest.delete("new");
    const q = rest.toString();
    window.history.replaceState(null, "", q ? `${pathname}?${q}` : pathname);
  }, [requested, isPending, wallets, params, pathname]);

  // Without a wallet there's nowhere to add to; the wallets page offers to create one.
  if (!isPending && !canAdd) return null;

  return (
    <>
      <Button className="hidden md:inline-flex" onClick={start} disabled={!canAdd}>
        <Plus strokeWidth={1.25} /> New transaction
      </Button>
      <Button
        size="icon-lg"
        aria-label="New transaction"
        className="fixed right-5 bottom-[calc(env(safe-area-inset-bottom)+1.25rem)] z-40 md:hidden"
        onClick={start}
        disabled={!canAdd}
      >
        <Plus strokeWidth={1.25} className="size-6" />
      </Button>
      <TransactionSheet
        open={open}
        onOpenChange={setOpen}
        wallet={byId.get(walletId)}
        editing={null}
        onWalletChange={setWalletId}
      />
    </>
  );
}
