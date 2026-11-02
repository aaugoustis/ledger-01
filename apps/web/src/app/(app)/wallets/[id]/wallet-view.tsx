"use client";

import { useInfiniteQuery, useMutation, useQuery } from "@connectrpc/connect-query";
import {
  OccurrenceScope,
  type Transaction,
  TransactionService,
  WalletService,
} from "@ledger-01/api";
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
import { Button } from "@ledger-01/ui/components/button";
import { Panel } from "@ledger-01/ui/components/panel";
import { Skeleton } from "@ledger-01/ui/components/skeleton";
import { ChevronLeft, ChevronRight, Plus, Trash2 } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useMemo, useState } from "react";
import { ScopeDialog } from "@/components/scope-dialog";
import { TransactionSheet } from "@/components/transaction-sheet";
import { useCategories, useCurrencies, useLabels, useMembers } from "@/lib/api/hooks";
import { formatPeriod, parsePeriod, shiftPeriod, today } from "@/lib/period";
import { type Filters, FiltersBar, filterInput, noFilters } from "./filters";
import { TransactionList } from "./transaction-list";
import { WalletHeader } from "./wallet-header";

export function WalletView({ walletId }: { walletId: string }) {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const period = parsePeriod(params);

  const walletQ = useQuery(WalletService.method.getWallet, { id: walletId, today: today() });
  const wallet = walletQ.data?.wallet;
  const { categories, byId: categoriesById } = useCategories();
  const members = useMembers();
  const people = members.shown ? members.people : [];
  const { byId: labelsById } = useLabels();
  const { minorUnits } = useCurrencies();

  const [filters, setFilters] = useState<Filters>(noFilters);
  const units = minorUnits(wallet?.currency ?? "USD");
  const filter = useMemo(
    () => filterInput(walletId, period, filters, units),
    [walletId, period, filters, units],
  );

  const summaryQ = useQuery(TransactionService.method.getPeriodSummary, { filter });
  const listQ = useInfiniteQuery(
    TransactionService.method.listTransactions,
    { filter, pageSize: 50, pageToken: "" },
    { pageParamKey: "pageToken", getNextPageParam: (last) => last.nextPageToken || undefined },
  );
  const transactions = useMemo(
    () => listQ.data?.pages.flatMap((p) => p.transactions) ?? [],
    [listQ.data],
  );

  const [editor, setEditor] = useState<{ open: boolean; editing: Transaction | null }>({
    open: false,
    editing: null,
  });
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [confirmDelete, setConfirmDelete] = useState(false);
  const del = useMutation(TransactionService.method.deleteTransactions);

  function goTo(dir: 1 | -1) {
    const p = shiftPeriod(period, dir);
    router.replace(`${pathname}?from=${p.from}&to=${p.to}`, { scroll: false });
  }

  // Deleting occurrences of recurring transactions asks whether the later ones go too.
  const selectedRecurring = transactions.some((t) => selected.has(t.id) && t.recurringRuleId);

  async function deleteSelected(scope = OccurrenceScope.UNSPECIFIED) {
    await del.mutateAsync({ ids: [...selected], scope, today: today() });
    setSelected(new Set());
    setConfirmDelete(false);
  }

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

  const summary = summaryQ.data;
  const tiles = [
    { label: "Wallet balance", value: summary?.balanceMinor, sign: "always" as const },
    { label: "Period change", value: summary?.changeMinor, sign: "always" as const },
    {
      label: "Period expenses",
      value: summary ? -summary.expensesMinor : undefined,
      sign: "auto" as const,
    },
    { label: "Period income", value: summary?.incomeMinor, sign: "always" as const },
  ];

  return (
    <div className="flex flex-col gap-8">
      <WalletHeader
        walletId={walletId}
        wallet={wallet}
        actions={
          <Button
            className="hidden md:inline-flex"
            onClick={() => setEditor({ open: true, editing: null })}
            disabled={!wallet}
          >
            <Plus strokeWidth={1.25} /> New transaction
          </Button>
        }
      />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <Button
            variant="secondary"
            size="icon-sm"
            aria-label="Previous period"
            onClick={() => goTo(-1)}
          >
            <ChevronLeft strokeWidth={1.25} />
          </Button>
          <span className="tabular min-w-56 text-center text-hud-num text-ink">
            {formatPeriod(period)}
          </span>
          <Button
            variant="secondary"
            size="icon-sm"
            aria-label="Next period"
            onClick={() => goTo(1)}
          >
            <ChevronRight strokeWidth={1.25} />
          </Button>
        </div>
        <FiltersBar
          value={filters}
          onChange={setFilters}
          categories={categories}
          people={people}
          currency={wallet?.currency ?? ""}
        />
      </div>

      <dl className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {tiles.map((t) => (
          <Panel key={t.label} notch="sm" bodyClassName="p-4 sm:p-5">
            <dt className="hud-sm text-ink-soft">{t.label}</dt>
            <dd className="mt-2">
              {t.value === undefined || !wallet ? (
                <Skeleton className="h-7 w-32" />
              ) : (
                <Amount
                  minor={t.value}
                  currency={wallet.currency}
                  minorUnits={units}
                  sign={t.sign}
                  size="lg"
                  className="max-sm:text-amount-md"
                />
              )}
            </dd>
          </Panel>
        ))}
      </dl>

      {selected.size > 0 && (
        <div className="glass sticky top-24 z-30 flex items-center justify-between rounded-full py-2 pr-2 pl-5">
          <span className="text-label text-ink">{selected.size} selected</span>
          <div className="flex gap-2">
            <Button variant="ghost" size="sm" onClick={() => setSelected(new Set())}>
              Clear
            </Button>
            <Button variant="destructive" size="sm" onClick={() => setConfirmDelete(true)}>
              <Trash2 strokeWidth={1.25} /> Delete
            </Button>
          </div>
        </div>
      )}

      {wallet && (
        <TransactionList
          wallet={wallet}
          transactions={transactions}
          loading={listQ.isPending}
          categoriesById={categoriesById}
          labelsById={labelsById}
          peopleById={members.shown ? members.byId : undefined}
          minorUnits={minorUnits}
          selected={selected}
          onSelect={setSelected}
          onOpen={(t) => setEditor({ open: true, editing: t })}
          hasMore={Boolean(listQ.hasNextPage)}
          loadingMore={listQ.isFetchingNextPage}
          onLoadMore={() => listQ.fetchNextPage()}
        />
      )}

      {/* Mobile quick-add */}
      <Button
        size="icon-lg"
        aria-label="New transaction"
        className="fixed right-5 bottom-[calc(env(safe-area-inset-bottom)+1.25rem)] z-40 md:hidden"
        onClick={() => setEditor({ open: true, editing: null })}
        disabled={!wallet}
      >
        <Plus strokeWidth={1.25} className="size-6" />
      </Button>

      <TransactionSheet
        open={editor.open}
        onOpenChange={(open) => setEditor((e) => ({ ...e, open }))}
        wallet={wallet}
        editing={editor.editing}
      />

      <AlertDialog open={confirmDelete && !selectedRecurring} onOpenChange={setConfirmDelete}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {selected.size} transactions?</AlertDialogTitle>
            <AlertDialogDescription>
              Transfers are deleted from both wallets. This can't be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={() => deleteSelected()} disabled={del.isPending}>
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <ScopeDialog
        open={confirmDelete && selectedRecurring}
        onOpenChange={setConfirmDelete}
        title={`Delete ${selected.size} transactions?`}
        description="Some of them repeat. Delete only these, or these and the ones after them? Past ones stay."
        only="Only these"
        future="These and future ones"
        destructive
        disabled={del.isPending}
        onChoose={(scope) => deleteSelected(scope)}
      />
    </div>
  );
}
