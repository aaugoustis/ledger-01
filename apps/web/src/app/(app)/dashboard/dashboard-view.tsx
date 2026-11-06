"use client";

import { useQuery } from "@connectrpc/connect-query";
import { DashboardService } from "@ledger-01/api";
import { Amount } from "@ledger-01/ui/components/amount";
import { SectionHeader } from "@ledger-01/ui/components/hud";
import { Wallet as WalletIcon } from "lucide-react";
import Link from "next/link";
import { QuickAdd } from "@/components/quick-add";
import { useBaseCurrency, useCurrencies, useWallets } from "@/lib/api/hooks";
import { today } from "@/lib/period";
import { DashboardBody } from "./dashboard-body";
import { ReportFilters, useReportState } from "./report-filters";

export function DashboardView() {
  const base = useBaseCurrency();
  const { state, set, filter, bucket, currency, query } = useReportState(base);
  // Only an explicit currency is sent; the server reports in the base currency by default.
  const { data, isPending } = useQuery(DashboardService.method.getDashboard, {
    filter,
    bucket,
    today: today(),
    currency: currency === base ? "" : currency,
  });
  const shown = data?.currency ?? currency;
  const { minorUnits } = useCurrencies();
  const { wallets } = useWallets();

  return (
    <div className="flex flex-col gap-8">
      <SectionHeader code="S-00" label="Overview" title="Dashboard" aside={<QuickAdd />} />

      <ul className="flex gap-3 overflow-x-auto pb-1">
        {wallets
          .filter((w) => !w.hidden && !w.archived)
          .map((w) => (
            <li key={w.id} className="shrink-0">
              <Link
                href={`/wallets/${w.id}`}
                className="flex items-center gap-3 rounded-full bg-shell py-2 pr-4 pl-3 shadow-mold hover:bg-shell-recess"
              >
                <WalletIcon className="size-4 text-ink-soft" strokeWidth={1.25} />
                <span className="text-label text-ink">{w.name}</span>
                <Amount
                  minor={w.balanceMinor}
                  currency={w.currency}
                  minorUnits={minorUnits(w.currency)}
                  size="sm"
                />
              </Link>
            </li>
          ))}
      </ul>

      <ReportFilters state={state} set={set} currency={shown} />

      <DashboardBody
        data={data}
        isPending={isPending}
        bucket={state.bucket}
        currency={shown}
        drillQuery={query}
      />
    </div>
  );
}
