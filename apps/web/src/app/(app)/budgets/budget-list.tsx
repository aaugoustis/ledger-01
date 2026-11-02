"use client";

import { useQuery } from "@connectrpc/connect-query";
import { BudgetService, type BudgetSummary } from "@ledger-01/api";
import { Amount } from "@ledger-01/ui/components/amount";
import { BudgetMeter } from "@ledger-01/ui/components/budget-meter";
import { Button } from "@ledger-01/ui/components/button";
import { SectionHeader } from "@ledger-01/ui/components/hud";
import { Panel } from "@ledger-01/ui/components/panel";
import { Skeleton } from "@ledger-01/ui/components/skeleton";
import { formatMoney } from "@ledger-01/ui/lib/money";
import { Plus } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useCurrencies } from "@/lib/api/hooks";
import { periodStatus, repeatShort, todayShare } from "@/lib/budgets";
import { formatPeriod, today } from "@/lib/period";
import { BudgetEditor } from "./budget-editor";

export const budgetCode = (i: number) => `B-${String(i + 1).padStart(2, "0")}`;

export function BudgetList() {
  const router = useRouter();
  const { data, isPending } = useQuery(BudgetService.method.listBudgets, { today: today() });
  const [creating, setCreating] = useState(false);
  const budgets = data?.budgets ?? [];

  return (
    <div className="flex flex-col gap-10">
      <SectionHeader
        code="S-02"
        label="Budgets"
        title="Budgets"
        ruler
        aside={
          <Button onClick={() => setCreating(true)}>
            <Plus strokeWidth={1.25} /> Add budget
          </Button>
        }
      />

      {data && data.unconvertedCurrencies.length > 0 && (
        <p className="rounded-md bg-shell-recess px-4 py-2 text-body-sm text-ink">
          Expenses in {data.unconvertedCurrencies.join(", ")} aren't counted: there's no transfer to
          take a rate from yet.
        </p>
      )}

      {isPending ? (
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} className="h-52 rounded-lg" />
          ))}
        </div>
      ) : budgets.length === 0 ? (
        <Panel recess notch="md" className="max-w-xl">
          <div className="flex flex-col items-start gap-4">
            <p className="text-body-md text-ink">
              No budgets yet. Set a limit for a month of groceries, a trip, or all your spending.
            </p>
            <Button onClick={() => setCreating(true)}>
              <Plus strokeWidth={1.25} /> Add budget
            </Button>
          </div>
        </Panel>
      ) : (
        <ul className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {budgets.map((s, i) => (
            <li key={s.budget?.id}>
              <BudgetCard summary={s} index={i} />
            </li>
          ))}
        </ul>
      )}

      <BudgetEditor
        open={creating}
        onOpenChange={setCreating}
        editing={null}
        onSaved={(b) => router.push(`/budgets/${b.id}`)}
      />
    </div>
  );
}

function BudgetCard({ summary, index }: { summary: BudgetSummary; index: number }) {
  const { minorUnits } = useCurrencies();
  const b = summary.budget;
  const p = summary.current;
  if (!b || !p) return null;
  const units = minorUnits(b.currency);
  const over = p.remainingMinor < 0n;
  return (
    <Link
      href={`/budgets/${b.id}`}
      className="group block rounded-lg outline-none focus-visible:ring-[3px] focus-visible:ring-ring/60"
    >
      <Panel
        serial={`${budgetCode(index)} · ${repeatShort(b.repeat).toUpperCase()}`}
        hinge={index % 3 === 0 ? "24px" : index % 3 === 1 ? "45%" : "70%"}
        className="transition-transform group-hover:-translate-y-0.5"
      >
        <div className="flex flex-col gap-4">
          <div className="flex items-baseline justify-between gap-3">
            <h2 className="min-w-0 font-display text-display-sm break-words text-ink">{b.name}</h2>
            <span className="hud-sm shrink-0 text-ink-soft">{periodStatus(p)}</span>
          </div>
          <div className="flex items-baseline gap-2">
            <Amount
              minor={p.spentMinor}
              currency={b.currency}
              minorUnits={units}
              size="lg"
              sign="never"
              tone="plain"
              code={false}
            />
            <span className="text-body-sm text-ink-soft">
              of {formatMoney(b.amountMinor, b.currency, units)}
            </span>
          </div>
          <BudgetMeter
            spent={p.spentMinor}
            budget={b.amountMinor}
            elapsed={todayShare(p)}
            label={`${b.name} budget`}
          />
          <div className="flex items-baseline justify-between gap-3 text-body-sm">
            <span className={over ? "text-expense-ink" : "text-ink"}>
              {over
                ? `Over by ${formatMoney(-p.remainingMinor, b.currency, units)}`
                : `${formatMoney(p.remainingMinor, b.currency, units)} left`}
            </span>
            {p.dailyAllowanceMinor > 0n && (
              <span className="tabular text-ink-soft">
                {formatMoney(p.dailyAllowanceMinor, b.currency, units, { code: false })}/day
              </span>
            )}
          </div>
          <span className="hud-sm text-ink-soft">
            {p.period && formatPeriod({ from: p.period.from, to: p.period.to })}
          </span>
        </div>
      </Panel>
    </Link>
  );
}
