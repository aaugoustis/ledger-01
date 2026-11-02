"use client";

import { useMutation, useQuery } from "@connectrpc/connect-query";
import { type BudgetProgress, BudgetService, BudgetState } from "@ledger-01/api";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@ledger-01/ui/components/alert-dialog";
import { Amount } from "@ledger-01/ui/components/amount";
import { BudgetMeter } from "@ledger-01/ui/components/budget-meter";
import { Button } from "@ledger-01/ui/components/button";
import { CategoryIcon } from "@ledger-01/ui/components/category-icon";
import { DonutChart } from "@ledger-01/ui/components/charts";
import { HudLabel } from "@ledger-01/ui/components/hud";
import { Panel } from "@ledger-01/ui/components/panel";
import { Skeleton } from "@ledger-01/ui/components/skeleton";
import { categoryColors } from "@ledger-01/ui/lib/category-icons";
import { formatMoney } from "@ledger-01/ui/lib/money";
import { cn } from "@ledger-01/ui/lib/utils";
import { ChevronLeft, ChevronRight, Pencil, Trash2 } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";
import { useCategories, useCurrencies, useWallets } from "@/lib/api/hooks";
import { periodStatus, repeatOptions, todayShare } from "@/lib/budgets";
import { formatDay, formatPeriod, today } from "@/lib/period";
import { BudgetEditor } from "../budget-editor";

const range = (p: BudgetProgress) => (p.period ? formatPeriod(p.period) : "");

/** Period start and end under the meter, with "Today" under the meter's today tick. */
function TodayAxis({ progress }: { progress: BudgetProgress }) {
  const at = todayShare(progress);
  // Near the edges the caption hangs off the tick toward the middle so it stays inside.
  const shift = at === undefined ? "" : at < 0.15 ? "0%" : at > 0.85 ? "-100%" : "-50%";
  return (
    <div
      className={cn(
        "hud-sm relative flex justify-between text-ink-soft",
        at !== undefined && "mb-5",
      )}
    >
      <span>{progress.period && formatDay(progress.period.from)}</span>
      <span>{progress.period && formatDay(progress.period.to)}</span>
      {at !== undefined && (
        <span
          className="absolute top-full mt-1 whitespace-nowrap text-ink"
          style={{ left: `${at * 100}%`, transform: `translateX(${shift})` }}
        >
          ◆ Today
        </span>
      )}
    </div>
  );
}

export function BudgetDetail({ budgetId }: { budgetId: string }) {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const periodDate = params.get("period") ?? "";
  const { data, isPending, isError } = useQuery(BudgetService.method.getBudget, {
    id: budgetId,
    today: today(),
    periodDate,
  });
  const { byId: categories } = useCategories();
  const { wallets, byId: walletsById } = useWallets();
  const { minorUnits } = useCurrencies();
  const [editing, setEditing] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const del = useMutation(BudgetService.method.deleteBudget);

  if (isError) {
    return (
      <Panel recess className="max-w-xl">
        <p className="text-body-md text-ink">This budget doesn't exist or isn't yours.</p>
      </Panel>
    );
  }

  const b = data?.budget;
  const p = data?.progress;
  const currency = b?.currency ?? "USD";
  const units = minorUnits(currency);
  const money = { currency, minorUnits: units };
  const history = data?.history ?? [];
  const at = p ? history.findIndex((h) => h.period?.from === p.period?.from) : -1;
  const older = at >= 0 ? history[at + 1] : undefined;
  const newer = at > 0 ? history[at - 1] : undefined;
  const goTo = (h: BudgetProgress | undefined) => {
    if (!h?.period) return;
    const q = new URLSearchParams(params);
    if (h === history[0]) q.delete("period");
    else q.set("period", h.period.from);
    router.replace(q.size ? `${pathname}?${q}` : pathname, { scroll: false });
  };
  const over = p ? p.remainingMinor < 0n : false;
  const repeatLabel = repeatOptions.find((o) => o.value === b?.repeat)?.label ?? "";

  // Wallets keep the palette color of their position in the wallet list, so a wallet looks
  // the same in every period.
  const walletColor = (id: string) => {
    const i = wallets.findIndex((w) => w.id === id);
    return categoryColors[Math.max(i, 0) % categoryColors.length] ?? "cat-slate";
  };

  const tiles = [
    {
      label: "Budgeted",
      node: b && (
        <Amount
          minor={b.amountMinor}
          currency={currency}
          minorUnits={units}
          size="lg"
          sign="never"
          tone="plain"
          className="max-sm:text-amount-md"
        />
      ),
    },
    {
      label: "Spent",
      node: p && (
        <Amount
          minor={-p.spentMinor}
          currency={currency}
          minorUnits={units}
          size="lg"
          sign="auto"
          className="max-sm:text-amount-md"
        />
      ),
    },
    {
      label: over ? "Over budget" : "Left",
      node: p && (
        <Amount
          minor={p.remainingMinor}
          currency={currency}
          minorUnits={units}
          size="lg"
          sign="auto"
          tone={over ? "auto" : "plain"}
          className="max-sm:text-amount-md"
        />
      ),
    },
    {
      label: p?.state === BudgetState.UPCOMING ? "Per day, once it starts" : "Can spend per day",
      node: p && (
        <Amount
          minor={p.dailyAllowanceMinor}
          currency={currency}
          minorUnits={units}
          size="lg"
          sign="never"
          tone="plain"
          className="max-sm:text-amount-md"
        />
      ),
    },
  ];

  const categoryTotal = (data?.categories ?? []).reduce((s, t) => s + t.amountMinor, 0n);
  const walletTotal = (data?.wallets ?? []).reduce((s, t) => s + t.amountMinor, 0n);
  const share = (v: bigint, total: bigint) =>
    total > 0n ? `${(Number((v * 1000n) / total) / 10).toFixed(1)}%` : "";

  return (
    <div className="flex flex-col gap-8">
      <div className="flex flex-col gap-3">
        <HudLabel code="S-02">
          <Link href="/budgets" className="hover:text-ink">
            Budgets
          </Link>
        </HudLabel>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex min-w-0 flex-col gap-1">
            <h1 className="truncate font-display text-display-md text-ink sm:text-display-lg">
              {b?.name ?? <Skeleton className="h-10 w-48" />}
            </h1>
            {b && (
              <span className="hud-sm text-ink-soft">
                {repeatLabel} · {formatMoney(b.amountMinor, currency, units)}
              </span>
            )}
          </div>
          <div className="flex gap-2">
            <Button variant="secondary" onClick={() => setEditing(true)} disabled={!b}>
              <Pencil strokeWidth={1.25} /> Edit
            </Button>
            <Button
              variant="ghost"
              size="icon"
              aria-label="Delete budget"
              onClick={() => setConfirmDelete(true)}
              disabled={!b}
            >
              <Trash2 strokeWidth={1.25} />
            </Button>
          </div>
        </div>
      </div>

      <div className="flex items-center gap-2">
        <Button
          variant="secondary"
          size="icon-sm"
          aria-label="Previous period"
          disabled={!older}
          onClick={() => goTo(older)}
        >
          <ChevronLeft strokeWidth={1.25} />
        </Button>
        <span className="tabular min-w-56 text-center text-hud-num text-ink">
          {p ? range(p) : ""}
        </span>
        <Button
          variant="secondary"
          size="icon-sm"
          aria-label="Next period"
          disabled={!newer}
          onClick={() => goTo(newer)}
        >
          <ChevronRight strokeWidth={1.25} />
        </Button>
        {p && <span className="hud-sm ml-2 text-ink-soft">{periodStatus(p)}</span>}
      </div>

      {data && data.unconvertedCurrencies.length > 0 && (
        <p className="rounded-md bg-shell-recess px-4 py-2 text-body-sm text-ink">
          Expenses in {data.unconvertedCurrencies.join(", ")} aren't counted: there's no transfer to
          take a rate from yet.
        </p>
      )}

      <dl className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {tiles.map((t) => (
          <Panel key={t.label} notch="sm" bodyClassName="p-4 sm:p-5">
            <dt className="hud-sm text-ink-soft">{t.label}</dt>
            <dd className="mt-2">{t.node ?? <Skeleton className="h-7 w-28" />}</dd>
          </Panel>
        ))}
      </dl>

      <Panel serial="BGT-01 · PROGRESS" cross bodyClassName="p-5 pt-11 sm:p-6 sm:pt-11">
        {isPending || !b || !p ? (
          <Skeleton className="h-16" />
        ) : (
          <div className="flex flex-col gap-3">
            <div className="flex flex-wrap items-baseline justify-between gap-2 text-body-sm">
              <span className="text-ink">
                {formatMoney(p.spentMinor, currency, units)} of{" "}
                {formatMoney(b.amountMinor, currency, units)} spent
              </span>
              <span className={over ? "text-expense-ink" : "text-ink-soft"}>
                {over
                  ? `Over by ${formatMoney(-p.remainingMinor, currency, units)}`
                  : `${formatMoney(p.remainingMinor, currency, units)} left`}
              </span>
            </div>
            <BudgetMeter
              spent={p.spentMinor}
              budget={b.amountMinor}
              elapsed={todayShare(p)}
              label={`${b.name} budget`}
            />
            <TodayAxis progress={p} />
          </div>
        )}
      </Panel>

      <div className="grid gap-6 *:min-w-0 lg:grid-cols-2">
        <Panel serial="BGT-02 · CATEGORIES" bodyClassName="p-5 pt-11 sm:p-6 sm:pt-11">
          <div className="flex flex-col gap-5">
            <h2 className="font-display text-display-sm text-ink">Spending by category</h2>
            {data && data.categories.length === 0 ? (
              <p className="text-body-sm text-ink-soft">No expenses in this period.</p>
            ) : (
              <>
                <DonutChart
                  money={money}
                  centerLabel="Spent"
                  slices={(data?.categories ?? []).map((t) => {
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
                  {(data?.categories ?? []).map((t) => {
                    const c = categories.get(t.categoryId);
                    return (
                      <li
                        key={t.categoryId}
                        className="flex items-center gap-3 border-b border-seam/60 py-2.5 last:border-b-0"
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
                          {t.transactionCount.toString()} txn ·{" "}
                          {share(t.amountMinor, categoryTotal)}
                        </span>
                        <Amount
                          minor={-t.amountMinor}
                          currency={currency}
                          minorUnits={units}
                          code={false}
                        />
                      </li>
                    );
                  })}
                </ul>
              </>
            )}
          </div>
        </Panel>

        <Panel serial="BGT-03 · WALLETS" bodyClassName="p-5 pt-11 sm:p-6 sm:pt-11">
          <div className="flex flex-col gap-5">
            <h2 className="font-display text-display-sm text-ink">Spending by wallet</h2>
            {data && data.wallets.length === 0 ? (
              <p className="text-body-sm text-ink-soft">No expenses in this period.</p>
            ) : (
              <>
                <DonutChart
                  money={money}
                  centerLabel="Spent"
                  slices={(data?.wallets ?? []).map((t) => ({
                    key: t.walletId,
                    label: walletsById.get(t.walletId)?.name ?? "Wallet",
                    color: walletColor(t.walletId),
                    amount: t.amountMinor,
                  }))}
                />
                <ul className="flex flex-col">
                  {(data?.wallets ?? []).map((t) => {
                    const w = walletsById.get(t.walletId);
                    return (
                      <li
                        key={t.walletId}
                        className="flex items-center gap-3 border-b border-seam/60 py-2.5 last:border-b-0"
                      >
                        <span
                          className="size-3 shrink-0 rounded-full"
                          style={{ background: `var(--color-${walletColor(t.walletId)})` }}
                        />
                        <Link
                          href={`/wallets/${t.walletId}`}
                          className="min-w-0 flex-1 truncate text-label text-ink hover:underline"
                        >
                          {w?.name ?? "Wallet"}
                        </Link>
                        <span className="hud-sm hidden text-ink-soft sm:inline">
                          {w?.currency} · {share(t.amountMinor, walletTotal)}
                        </span>
                        <Amount
                          minor={-t.amountMinor}
                          currency={currency}
                          minorUnits={units}
                          code={false}
                        />
                      </li>
                    );
                  })}
                </ul>
              </>
            )}
          </div>
        </Panel>
      </div>

      <div className="grid gap-6 *:min-w-0 lg:grid-cols-[2fr_3fr]">
        <Panel serial="BGT-04 · HISTORY" bodyClassName="p-0 pt-9 sm:p-0 sm:pt-9">
          <ul>
            {history.map((h, i) => {
              const selected = h.period?.from === p?.period?.from;
              return (
                <li key={h.period?.from ?? i} className="border-b border-seam/60 last:border-b-0">
                  <button
                    type="button"
                    onClick={() => goTo(h)}
                    aria-current={selected || undefined}
                    className={cn(
                      "flex w-full flex-col gap-2 px-4 py-3 text-left hover:bg-shell-recess/60 sm:px-6",
                      selected && "bg-shell-recess",
                    )}
                  >
                    <span className="flex items-baseline justify-between gap-3">
                      <span className="hud-sm min-w-0 truncate text-ink">{range(h)}</span>
                      <span
                        className={cn(
                          "tabular text-body-sm",
                          h.remainingMinor < 0n ? "text-expense-ink" : "text-ink-soft",
                        )}
                      >
                        {formatMoney(h.spentMinor, currency, units, { code: false })}
                      </span>
                    </span>
                    {b && (
                      <BudgetMeter
                        size="sm"
                        spent={h.spentMinor}
                        budget={b.amountMinor}
                        elapsed={todayShare(h)}
                        label={`Spent ${range(h)}`}
                      />
                    )}
                  </button>
                </li>
              );
            })}
          </ul>
        </Panel>

        <Panel serial="BGT-05 · TRANSACTIONS" bodyClassName="p-0 pt-9 sm:p-0 sm:pt-9">
          {data && data.transactions.length === 0 ? (
            <p className="px-6 pb-6 text-body-sm text-ink-soft">No expenses in this period.</p>
          ) : (
            <ul>
              {(data?.transactions ?? []).map((t) => {
                const c = t.categoryId ? categories.get(t.categoryId) : undefined;
                const w = walletsById.get(t.walletId);
                return (
                  <li
                    key={t.id}
                    className="flex items-center gap-3 border-b border-seam/60 px-4 py-3 last:border-b-0 sm:px-6"
                  >
                    <CategoryIcon
                      icon={c?.icon ?? "circle-help"}
                      color={c?.color ?? "cat-slate"}
                      size="sm"
                    />
                    <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                      <span className="truncate text-label text-ink">
                        {t.note || c?.name || "Expense"}
                      </span>
                      <span className="hud-sm truncate text-ink-soft">
                        {formatDay(t.occurredOn)} · {w?.name ?? "Wallet"}
                      </span>
                    </div>
                    <Amount
                      minor={-t.amountMinor}
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

      <BudgetEditor open={editing} onOpenChange={setEditing} editing={b ?? null} />

      <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {b?.name}?</AlertDialogTitle>
            <AlertDialogDescription>
              The budget and its history go away. Your transactions stay as they are.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={del.isPending}
              onClick={async () => {
                await del.mutateAsync({ id: budgetId });
                router.push("/budgets");
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
