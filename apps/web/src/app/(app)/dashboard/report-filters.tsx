"use client";

import { CategoryKind } from "@ledger-01/api";
import { Button } from "@ledger-01/ui/components/button";
import { CategoryIcon } from "@ledger-01/ui/components/category-icon";
import { Popover, PopoverContent, PopoverTrigger } from "@ledger-01/ui/components/popover";
import { ToggleGroup, ToggleGroupItem } from "@ledger-01/ui/components/toggle-group";
import { ChevronLeft, ChevronRight, SlidersHorizontal } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useMemo } from "react";
import { AmountRange } from "@/components/amount-range";
import { MultiPicker } from "@/components/multi-picker";
import { NoteSearch } from "@/components/note-search";
import { useCategories, useCurrencies, useWallets } from "@/lib/api/hooks";
import { type BucketName, bucketProto, periodForBucket } from "@/lib/buckets";
import { formatPeriod, shiftPeriod } from "@/lib/period";
import { amountBound, parseReportState, type ReportState, reportQuery } from "@/lib/report-state";

export type { ReportState };

/**
 * Report filters live in the URL so drill-downs and reloads keep them. The report currency is the
 * URL's `currency`, else `defaultCurrency` (the base currency, or a wallet's own).
 */
export function useReportState(defaultCurrency: string) {
  const { minorUnits } = useCurrencies();
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const state = useMemo(() => parseReportState(params, defaultCurrency), [params, defaultCurrency]);

  function set(next: Partial<ReportState>, path = pathname) {
    const s = { ...state, ...next };
    const q = reportQuery(s, params, Boolean(next.bucket));
    router.replace(`${path}?${q}`, { scroll: false });
  }

  const filter = {
    period: state.period,
    walletIds: state.walletIds,
    categoryIds: state.categoryIds,
    noteQuery: state.note,
    minAmountMinor: amountBound(state.min, minorUnits(state.currency)),
    maxAmountMinor: amountBound(state.max, minorUnits(state.currency)),
  };
  return {
    state,
    set,
    filter,
    bucket: bucketProto[state.bucket],
    currency: state.currency,
    query: params.toString(),
  };
}

export function ReportFilters({
  state,
  set,
  currency,
  showWallets = true,
  showCategories = true,
}: {
  state: ReportState;
  set: (next: Partial<ReportState>) => void;
  /** The report currency, for the amount bounds. */
  currency: string;
  showWallets?: boolean;
  showCategories?: boolean;
}) {
  const { wallets } = useWallets();
  const { categories } = useCategories();
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div className="flex items-center gap-2">
        <Button
          variant="secondary"
          size="icon-sm"
          aria-label="Previous period"
          onClick={() => set({ period: shiftPeriod(state.period, -1) })}
        >
          <ChevronLeft strokeWidth={1.25} />
        </Button>
        <span className="tabular min-w-56 text-center text-hud-num text-ink">
          {formatPeriod(state.period)}
        </span>
        <Button
          variant="secondary"
          size="icon-sm"
          aria-label="Next period"
          onClick={() => set({ period: shiftPeriod(state.period, 1) })}
        >
          <ChevronRight strokeWidth={1.25} />
        </Button>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <NoteSearch value={state.note} onChange={(note) => set({ note })} />
        <Popover>
          <PopoverTrigger asChild>
            <Button variant="secondary" size="sm">
              <SlidersHorizontal strokeWidth={1.25} /> Amount
              {state.min || state.max ? " · set" : ""}
            </Button>
          </PopoverTrigger>
          <PopoverContent align="end" className="flex w-80 flex-col gap-3">
            <AmountRange
              currency={currency}
              min={state.min}
              max={state.max}
              onChange={(r) => set(r)}
            />
            <p className="text-body-sm text-ink-soft">
              Amounts in {currency}; other currencies are converted first.
            </p>
            <Button
              variant="ghost"
              size="sm"
              className="self-start"
              onClick={() => set({ min: "", max: "" })}
            >
              Clear amounts
            </Button>
          </PopoverContent>
        </Popover>
        {showWallets && (
          <MultiPicker
            allLabel="All wallets"
            noun="wallets"
            searchPlaceholder="Search wallets…"
            value={state.walletIds}
            onChange={(walletIds) => set({ walletIds })}
            options={wallets
              .filter((w) => !w.archived)
              .map((w) => ({ key: w.id, label: `${w.name} · ${w.currency}` }))}
          />
        )}
        {showCategories && (
          <MultiPicker
            allLabel="All categories"
            noun="categories"
            searchPlaceholder="Search categories…"
            value={state.categoryIds}
            onChange={(categoryIds) => set({ categoryIds })}
            options={categories
              .filter((c) => !c.archived)
              .map((c) => ({
                key: c.id,
                label: c.name,
                group: c.kind === CategoryKind.INCOME ? "Income" : "Expense",
                icon: <CategoryIcon icon={c.icon} color={c.color} size="sm" />,
              }))}
          />
        )}
        <ToggleGroup
          type="single"
          variant="outline"
          size="sm"
          value={state.bucket}
          onValueChange={(v) => {
            if (!v) return;
            // The grouping picks the span too (a month by day, 3 months by week, 12 by month),
            // ending with the month currently shown; the arrows then step by that span.
            const bucket = v as BucketName;
            set({ bucket, period: periodForBucket(bucket, state.period.to) });
          }}
          aria-label="Group by"
        >
          <ToggleGroupItem value="day">Days</ToggleGroupItem>
          <ToggleGroupItem value="week">Weeks</ToggleGroupItem>
          <ToggleGroupItem value="month">Months</ToggleGroupItem>
        </ToggleGroup>
      </div>
    </div>
  );
}
