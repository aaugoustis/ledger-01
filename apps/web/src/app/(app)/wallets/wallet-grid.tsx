"use client";

import { Amount } from "@ledger-01/ui/components/amount";
import { Button } from "@ledger-01/ui/components/button";
import { Checkbox } from "@ledger-01/ui/components/checkbox";
import { SectionHeader } from "@ledger-01/ui/components/hud";
import { Label } from "@ledger-01/ui/components/label";
import { Panel } from "@ledger-01/ui/components/panel";
import { Skeleton } from "@ledger-01/ui/components/skeleton";
import { CalendarClock, EyeOff, Plus, Wallet as WalletIcon } from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import { useCurrencies, useWallets } from "@/lib/api/hooks";
import { CreateWalletDialog } from "./create-wallet-dialog";

export const walletCode = (i: number) => `W-${String(i + 1).padStart(2, "0")}`;

export function WalletGrid() {
  const [showHidden, setShowHidden] = useState(false);
  const { wallets, isPending } = useWallets({ includeHidden: true });
  const { minorUnits } = useCurrencies();
  const [creating, setCreating] = useState(false);
  const visible = wallets.filter((w) => showHidden || !w.hidden);
  const hiddenCount = wallets.filter((w) => w.hidden).length;

  return (
    <div className="flex flex-col gap-10">
      <SectionHeader
        code="S-01"
        label="Wallets"
        title="Wallets"
        ruler
        aside={
          <div className="flex gap-2">
            <Button variant="secondary" asChild>
              <Link href="/future">
                <CalendarClock strokeWidth={1.25} /> Future
              </Link>
            </Button>
            <Button onClick={() => setCreating(true)}>
              <Plus strokeWidth={1.25} /> Add wallet
            </Button>
          </div>
        }
      />

      {hiddenCount > 0 && (
        <div className="flex items-center gap-2">
          <Checkbox
            id="show-hidden"
            checked={showHidden}
            onCheckedChange={(v) => setShowHidden(v === true)}
          />
          <Label htmlFor="show-hidden" className="text-body-sm text-ink-soft">
            Show hidden wallets ({hiddenCount})
          </Label>
        </div>
      )}

      {isPending ? (
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} className="h-40 rounded-lg" />
          ))}
        </div>
      ) : visible.length === 0 ? (
        <Panel recess notch="md" className="max-w-xl">
          <div className="flex flex-col items-start gap-4">
            <p className="text-body-md text-ink">
              No wallets yet. Add one for each bank account, card or cash pile you track.
            </p>
            <Button onClick={() => setCreating(true)}>
              <Plus strokeWidth={1.25} /> Add wallet
            </Button>
          </div>
        </Panel>
      ) : (
        <ul className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {visible.map((w, i) => (
            <li key={w.id}>
              <Link
                href={`/wallets/${w.id}`}
                className="group block rounded-lg outline-none focus-visible:ring-[3px] focus-visible:ring-ring/60"
              >
                <Panel
                  serial={`${walletCode(i)} · ${w.currency}`}
                  hinge={i % 3 === 0 ? "24px" : i % 3 === 1 ? "45%" : "70%"}
                  cross
                  className="transition-transform group-hover:-translate-y-0.5"
                >
                  <div className="flex flex-col gap-4">
                    <div className="flex items-center gap-2">
                      <WalletIcon className="size-4 text-ink-soft" strokeWidth={1.25} />
                      <h2 className="truncate font-display text-display-sm text-ink">{w.name}</h2>
                      {w.hidden && (
                        <EyeOff
                          className="size-4 text-hud-mark"
                          strokeWidth={1.25}
                          aria-label="Hidden"
                        />
                      )}
                    </div>
                    <Amount
                      minor={w.balanceMinor}
                      currency={w.currency}
                      minorUnits={minorUnits(w.currency)}
                      size="lg"
                    />
                    <span className="hud-sm text-ink-soft">
                      {w.transactionCount.toString()} transactions
                    </span>
                  </div>
                </Panel>
              </Link>
            </li>
          ))}
        </ul>
      )}

      <CreateWalletDialog open={creating} onOpenChange={setCreating} />
    </div>
  );
}
