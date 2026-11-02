"use client";

import { useQuery } from "@connectrpc/connect-query";
import { CategoryKind, DashboardService, TransactionKind } from "@ledger-01/api";
import { Amount } from "@ledger-01/ui/components/amount";
import { CategoryIcon, ColorDot } from "@ledger-01/ui/components/category-icon";
import { FlowChart } from "@ledger-01/ui/components/charts";
import { HudLabel } from "@ledger-01/ui/components/hud";
import { Panel } from "@ledger-01/ui/components/panel";
import { Skeleton } from "@ledger-01/ui/components/skeleton";
import Link from "next/link";
import {
  useBaseCurrency,
  useCategories,
  useCurrencies,
  useLabels,
  useWallets,
} from "@/lib/api/hooks";
import { bucketLabels } from "@/lib/buckets";
import { formatDay, today } from "@/lib/period";
import { ReportFilters, useReportState } from "../../report-filters";

const WEEKDAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

export function CategoryReportView({ categoryId }: { categoryId: string }) {
  const base = useBaseCurrency();
  const { state, set, filter, bucket, currency: requested, query } = useReportState(base);
  const { data, isPending, isError } = useQuery(DashboardService.method.getCategoryReport, {
    categoryId,
    filter: { ...filter, categoryIds: [] },
    bucket,
    today: today(),
    currency: requested === base ? "" : requested,
  });
  const { byId: categories } = useCategories();
  const { byId: wallets } = useWallets();
  const { byId: labels } = useLabels();
  const { minorUnits } = useCurrencies();
  const category = categories.get(categoryId);
  const income = category?.kind === CategoryKind.INCOME;
  const currency = data?.currency ?? requested;
  const money = { currency, minorUnits: minorUnits(currency) };
  const sign = (v: bigint) => (income ? v : -v);

  if (isError) {
    return (
      <Panel recess className="max-w-xl">
        <p className="text-body-md text-ink">This category doesn't exist or isn't yours.</p>
      </Panel>
    );
  }

  const tiles = [
    {
      label: "Subtotal",
      node: data && (
        <Amount
          minor={sign(data.totalMinor)}
          currency={currency}
          minorUnits={money.minorUnits}
          size="lg"
          className="max-sm:text-amount-md"
        />
      ),
    },
    {
      label: "Transactions",
      node: data && (
        <span className="tabular text-amount-lg text-ink max-sm:text-amount-md">
          {data.transactionCount.toString()}
        </span>
      ),
    },
    {
      label: "Average",
      node: data && (
        <Amount
          minor={sign(data.averageMinor)}
          currency={currency}
          minorUnits={money.minorUnits}
          size="lg"
          className="max-sm:text-amount-md"
        />
      ),
    },
    {
      label: "Busiest day",
      node: data && (
        <span className="text-amount-lg text-ink max-sm:text-amount-md">
          {WEEKDAYS[data.busiestWeekday] ?? "—"}
        </span>
      ),
    },
  ];

  return (
    <div className="flex flex-col gap-8">
      <div className="flex flex-col gap-3">
        <HudLabel code="S-00">
          <Link href={`/dashboard?${query}`} className="hover:text-ink">
            Dashboard
          </Link>
        </HudLabel>
        <div className="flex items-center gap-3">
          {category && <CategoryIcon icon={category.icon} color={category.color} size="lg" />}
          <h1 className="font-display text-display-md text-ink sm:text-display-lg">
            {category?.name ?? <Skeleton className="h-10 w-48" />}
          </h1>
        </div>
      </div>

      <ReportFilters state={state} set={set} currency={currency} showCategories={false} />

      <dl className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {tiles.map((t) => (
          <Panel key={t.label} notch="sm" bodyClassName="p-4 sm:p-5">
            <dt className="hud-sm text-ink-soft">{t.label}</dt>
            <dd className="mt-2">{t.node ?? <Skeleton className="h-7 w-28" />}</dd>
          </Panel>
        ))}
      </dl>

      <Panel serial="CAT-01 · OVER TIME" cross bodyClassName="p-5 pt-11 sm:p-6 sm:pt-11">
        <div className="flex flex-col gap-4">
          <HudLabel rule={false}>
            {income ? "Income" : "Spending"} · {currency}
          </HudLabel>
          {isPending ? (
            <Skeleton className="h-56" />
          ) : (
            <FlowChart
              only={income ? "income" : "expense"}
              money={money}
              points={(data?.series ?? []).map((p) => ({
                ...bucketLabels(p.date, state.bucket),
                income: p.incomeMinor,
                expense: p.expenseMinor,
              }))}
            />
          )}
        </div>
      </Panel>

      <Panel serial="CAT-02 · TRANSACTIONS" bodyClassName="p-0 pt-9 sm:p-0 sm:pt-9">
        {data && data.transactions.length === 0 ? (
          <p className="px-6 pb-6 text-body-sm text-ink-soft">No transactions in this period.</p>
        ) : (
          <ul>
            {(data?.transactions ?? []).map((t) => {
              const w = wallets.get(t.walletId);
              const amount = t.kind === TransactionKind.EXPENSE ? -t.amountMinor : t.amountMinor;
              return (
                <li
                  key={t.id}
                  className="flex items-center gap-4 border-b border-seam/60 px-4 py-3 last:border-b-0 sm:px-6"
                >
                  {/* Phones stack date, wallet and note; wider screens put them in columns. */}
                  <div className="flex min-w-0 flex-1 flex-col gap-0.5 sm:flex-row sm:items-center sm:gap-4">
                    <span className="hud-sm shrink-0 text-ink-soft sm:w-28">
                      {formatDay(t.occurredOn)}
                    </span>
                    <Link
                      href={`/wallets/${t.walletId}`}
                      className="shrink-0 truncate text-label text-ink hover:underline sm:w-32"
                    >
                      {w?.name ?? "Wallet"}
                    </Link>
                    <span className="min-w-0 flex-1 truncate text-body-sm text-ink-soft">
                      {t.note}
                    </span>
                  </div>
                  <span className="hidden shrink-0 items-center gap-2 lg:flex">
                    {t.labelIds.map((id) => {
                      const l = labels.get(id);
                      return l ? (
                        <span
                          key={id}
                          className="flex items-center gap-1 text-body-sm text-ink-soft"
                        >
                          <ColorDot color={l.color} />
                          {l.name}
                        </span>
                      ) : null;
                    })}
                  </span>
                  <Amount
                    minor={amount}
                    currency={t.currency}
                    minorUnits={minorUnits(t.currency)}
                  />
                </li>
              );
            })}
          </ul>
        )}
      </Panel>
    </div>
  );
}
