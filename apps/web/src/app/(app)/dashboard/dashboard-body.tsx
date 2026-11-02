"use client";

import type { CategoryTotal, GetDashboardResponse } from "@ledger-01/api";
import { Amount } from "@ledger-01/ui/components/amount";
import { CategoryIcon } from "@ledger-01/ui/components/category-icon";
import { BalanceChart, DonutChart, FlowChart } from "@ledger-01/ui/components/charts";
import { HudLabel } from "@ledger-01/ui/components/hud";
import { Panel } from "@ledger-01/ui/components/panel";
import { Skeleton } from "@ledger-01/ui/components/skeleton";
import Link from "next/link";
import { useCategories, useCurrencies } from "@/lib/api/hooks";
import { type BucketName, bucketLabels } from "@/lib/buckets";

/**
 * The report itself, shared by the household dashboard and a wallet's overview: totals, balance
 * and cash-flow charts, category breakdowns linking to their drill-down (with `drillQuery`), and
 * transfers.
 */
export function DashboardBody({
  data,
  isPending,
  bucket,
  currency,
  drillQuery,
}: {
  data: GetDashboardResponse | undefined;
  isPending: boolean;
  bucket: BucketName;
  currency: string;
  drillQuery: string;
}) {
  const { minorUnits } = useCurrencies();
  const { byId: categories } = useCategories();
  const money = { currency, minorUnits: minorUnits(currency) };

  const points = (data?.series ?? []).map((p) => ({
    ...bucketLabels(p.date, bucket),
    balance: p.balanceMinor,
    income: p.incomeMinor,
    expense: p.expenseMinor,
  }));

  const tiles = [
    { label: "Total balance", value: data?.balanceMinor, sign: "always" as const },
    { label: "Period change", value: data?.changeMinor, sign: "always" as const },
    {
      label: "Period expenses",
      value: data ? -data.expensesMinor : undefined,
      sign: "auto" as const,
    },
    { label: "Period income", value: data?.incomeMinor, sign: "always" as const },
  ];

  const breakdown = (title: string, kind: "expense" | "income", totals: CategoryTotal[]) => {
    const total = totals.reduce((s, t) => s + t.amountMinor, 0n);
    return (
      <Panel
        serial={kind === "expense" ? "DSH-04 · EXPENSES" : "DSH-05 · INCOME"}
        bodyClassName="p-5 pt-11 sm:p-6 sm:pt-11"
      >
        <div className="flex flex-col gap-5">
          <h2 className="font-display text-display-sm text-ink">{title}</h2>
          {totals.length === 0 ? (
            <p className="text-body-sm text-ink-soft">
              No {kind === "expense" ? "expenses" : "income"} in this period.
            </p>
          ) : (
            <>
              <DonutChart
                money={money}
                centerLabel={kind === "expense" ? "Spent" : "Earned"}
                slices={totals.map((t) => {
                  const c = categories.get(t.categoryId);
                  return {
                    key: t.categoryId,
                    label: c?.name ?? "Category",
                    color: c?.color ?? "cat-slate",
                    icon: c?.icon ?? "circle-help",
                    amount: t.amountMinor,
                  };
                })}
              />
              <ul className="flex flex-col">
                {totals.map((t) => {
                  const c = categories.get(t.categoryId);
                  const share = total > 0n ? Number((t.amountMinor * 1000n) / total) / 10 : 0;
                  return (
                    <li key={t.categoryId} className="border-b border-seam/60 last:border-b-0">
                      <Link
                        href={`/dashboard/category/${t.categoryId}?${drillQuery}`}
                        className="flex items-center gap-3 py-2.5 hover:bg-shell-recess/60"
                      >
                        <CategoryIcon
                          icon={c?.icon ?? "circle-help"}
                          color={c?.color ?? "cat-slate"}
                          size="sm"
                        />
                        <span className="min-w-0 flex-1 truncate text-label text-ink">
                          {c?.name ?? "Category"}
                        </span>
                        <span className="hud-sm hidden text-ink-soft sm:inline">
                          {t.transactionCount.toString()} txn · {share.toFixed(1)}%
                        </span>
                        <Amount
                          minor={kind === "expense" ? -t.amountMinor : t.amountMinor}
                          currency={currency}
                          minorUnits={money.minorUnits}
                          code={false}
                        />
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </>
          )}
        </div>
      </Panel>
    );
  };

  return (
    <>
      {data && data.unconvertedCurrencies.length > 0 && (
        <p className="flex items-center gap-2 rounded-md bg-shell-recess px-4 py-2 text-body-sm text-ink">
          <span className="hud-sm text-ink-soft">Note</span>
          Not counted: {data.unconvertedCurrencies.join(", ")}. Totals use your own transfer rates,
          and there's no transfer between{" "}
          {data.unconvertedCurrencies.length > 1 ? "these currencies" : "this currency"} and{" "}
          {currency} yet.
        </p>
      )}

      <dl className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {tiles.map((t) => (
          <Panel key={t.label} notch="sm" bodyClassName="p-4 sm:p-5">
            <dt className="hud-sm text-ink-soft">{t.label}</dt>
            <dd className="mt-2">
              {t.value === undefined ? (
                <Skeleton className="h-7 w-32" />
              ) : (
                <Amount
                  minor={t.value}
                  currency={currency}
                  minorUnits={money.minorUnits}
                  sign={t.sign}
                  size="lg"
                  className="max-sm:text-amount-md"
                />
              )}
            </dd>
          </Panel>
        ))}
      </dl>

      <div className="grid gap-6 lg:grid-cols-2">
        <Panel serial="DSH-01 · BALANCE" cross bodyClassName="p-5 pt-11 sm:p-6 sm:pt-11">
          <div className="flex flex-col gap-4">
            <HudLabel rule={false}>Balance · {currency}</HudLabel>
            {isPending ? (
              <Skeleton className="h-56" />
            ) : (
              <BalanceChart points={points} money={money} />
            )}
          </div>
        </Panel>
        <Panel serial="DSH-02 · CASH FLOW" cross bodyClassName="p-5 pt-11 sm:p-6 sm:pt-11">
          <div className="flex flex-col gap-4">
            <HudLabel rule={false}>Income and expenses · {currency}</HudLabel>
            {isPending ? (
              <Skeleton className="h-56" />
            ) : (
              <FlowChart points={points} money={money} />
            )}
          </div>
        </Panel>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {breakdown("Expenses by category", "expense", data?.expenseCategories ?? [])}
        <div className="flex flex-col gap-6">
          {breakdown("Income by category", "income", data?.incomeCategories ?? [])}
          <Panel serial="DSH-03 · TRANSFERS" notch="sm" bodyClassName="p-5 pt-11 sm:p-6 sm:pt-11">
            <ul className="flex flex-col">
              {[
                {
                  label: "Transfers in",
                  count: data?.transfersInCount ?? 0n,
                  amount: data?.transfersInMinor ?? 0n,
                },
                {
                  label: "Transfers out",
                  count: data?.transfersOutCount ?? 0n,
                  amount: -(data?.transfersOutMinor ?? 0n),
                },
              ].map((r) => (
                <li
                  key={r.label}
                  className="flex items-center gap-3 border-b border-seam/60 py-2.5 last:border-b-0"
                >
                  <CategoryIcon icon="arrow-left-right" color="cat-slate" size="sm" />
                  <span className="flex-1 text-label text-ink">{r.label}</span>
                  <span className="hud-sm text-ink-soft">{r.count.toString()} txn</span>
                  <Amount
                    minor={r.amount}
                    currency={currency}
                    minorUnits={money.minorUnits}
                    code={false}
                  />
                </li>
              ))}
            </ul>
          </Panel>
        </div>
      </div>
    </>
  );
}
