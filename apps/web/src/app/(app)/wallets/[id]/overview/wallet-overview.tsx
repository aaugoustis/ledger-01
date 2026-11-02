"use client";

import { useQuery } from "@connectrpc/connect-query";
import { DashboardService, WalletService } from "@ledger-01/api";
import { Panel } from "@ledger-01/ui/components/panel";
import Link from "next/link";
import { DashboardBody } from "@/app/(app)/dashboard/dashboard-body";
import { ReportFilters, useReportState } from "@/app/(app)/dashboard/report-filters";
import { today } from "@/lib/period";
import { WalletHeader } from "../wallet-header";

/** The dashboard for one wallet, in the wallet's own currency. */
export function WalletOverview({ walletId }: { walletId: string }) {
  const walletQ = useQuery(WalletService.method.getWallet, { id: walletId, today: today() });
  const wallet = walletQ.data?.wallet;
  const currency = wallet?.currency ?? "";
  const { state, set, filter, bucket, query } = useReportState(currency);
  const { data, isPending } = useQuery(
    DashboardService.method.getDashboard,
    { filter: { ...filter, walletIds: [walletId] }, bucket, today: today(), currency },
    { enabled: Boolean(wallet) },
  );
  // Category drill-downs keep this wallet and its currency.
  const drill = new URLSearchParams(query);
  drill.set("wallets", walletId);
  if (currency) drill.set("currency", currency);

  if (walletQ.isError) {
    return (
      <Panel recess className="max-w-xl">
        <p className="text-body-md text-ink">This wallet doesn't exist or isn't yours.</p>
        <Link
          href="/wallets"
          className="mt-4 inline-block text-signal-ink underline-offset-4 hover:underline"
        >
          Back to wallets
        </Link>
      </Panel>
    );
  }

  return (
    <div className="flex flex-col gap-8">
      <WalletHeader walletId={walletId} wallet={wallet} />
      <ReportFilters state={state} set={set} currency={currency} showWallets={false} />
      <DashboardBody
        data={data}
        isPending={isPending || !wallet}
        bucket={state.bucket}
        currency={currency || "USD"}
        drillQuery={drill.toString()}
      />
    </div>
  );
}
