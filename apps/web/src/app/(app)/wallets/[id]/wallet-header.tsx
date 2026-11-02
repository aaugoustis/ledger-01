"use client";

import type { Wallet } from "@ledger-01/api";
import { Button } from "@ledger-01/ui/components/button";
import { HudLabel } from "@ledger-01/ui/components/hud";
import { Skeleton } from "@ledger-01/ui/components/skeleton";
import { cn } from "@ledger-01/ui/lib/utils";
import { Settings, Upload } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";

/**
 * A wallet's page header: name, Import and settings, and the tabs between its transactions, its
 * overview (the dashboard for this wallet alone) and its upcoming transactions.
 */
export function WalletHeader({
  walletId,
  wallet,
  actions,
}: {
  walletId: string;
  wallet: Wallet | undefined;
  /** Extra buttons before Import (e.g. "New transaction"). */
  actions?: React.ReactNode;
}) {
  const path = usePathname();
  const base = `/wallets/${walletId}`;
  const tabs = [
    { href: base, label: "Transactions" },
    { href: `${base}/overview`, label: "Overview" },
    { href: `${base}/future`, label: "Future" },
  ];
  return (
    <div className="flex flex-col gap-3">
      <HudLabel code="S-02" end={wallet?.currency}>
        <Link href="/wallets" className="hover:text-ink">
          Wallets
        </Link>
      </HudLabel>
      <div className="flex flex-wrap items-end justify-between gap-4">
        <h1 className="font-display text-display-md text-ink sm:text-display-lg">
          {wallet?.name ?? <Skeleton className="h-10 w-48" />}
        </h1>
        <div className="flex gap-2">
          {actions}
          <Button variant="secondary" asChild>
            <Link href={`${base}/import`}>
              <Upload strokeWidth={1.25} /> Import
            </Link>
          </Button>
          <Button variant="secondary" size="icon" asChild aria-label="Wallet settings">
            <Link href={`${base}/settings`}>
              <Settings strokeWidth={1.25} />
            </Link>
          </Button>
        </div>
      </div>
      <nav aria-label="Wallet views" className="flex gap-1 overflow-x-auto pt-2">
        {tabs.map((t) => (
          <Link
            key={t.href}
            href={t.href}
            aria-current={path === t.href ? "page" : undefined}
            className={cn(
              "rounded-full px-4 py-2 text-label whitespace-nowrap text-ink hover:bg-shell-recess",
              path === t.href && "bg-shell text-signal-ink shadow-mold",
            )}
          >
            {t.label}
          </Link>
        ))}
      </nav>
    </div>
  );
}
