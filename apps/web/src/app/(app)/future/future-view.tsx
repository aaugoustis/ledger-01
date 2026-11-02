"use client";

import { useQuery } from "@connectrpc/connect-query";
import {
  type Member,
  type Transaction,
  TransactionKind,
  TransactionService,
  WalletService,
} from "@ledger-01/api";
import { Amount } from "@ledger-01/ui/components/amount";
import { Button } from "@ledger-01/ui/components/button";
import { CategoryIcon } from "@ledger-01/ui/components/category-icon";
import { SectionHeader } from "@ledger-01/ui/components/hud";
import { Panel } from "@ledger-01/ui/components/panel";
import { Popover, PopoverContent, PopoverTrigger } from "@ledger-01/ui/components/popover";
import { Skeleton } from "@ledger-01/ui/components/skeleton";
import { Users } from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import { PeopleFilter } from "@/app/(app)/wallets/[id]/filters";
import { signed } from "@/app/(app)/wallets/[id]/transaction-list";
import { WalletHeader } from "@/app/(app)/wallets/[id]/wallet-header";
import { MemberAvatar } from "@/components/member-avatar";
import { RepeatMarker } from "@/components/repeat-marker";
import { TransactionSheet } from "@/components/transaction-sheet";
import { useCategories, useCurrencies, useMembers, useWallets } from "@/lib/api/hooks";
import { formatDay, today } from "@/lib/period";

/**
 * Upcoming transactions, soonest first and grouped by day: one wallet's, or the household's when
 * walletId is empty (each transfer once). These are pre-created recurring occurrences and
 * future-dated entries; none of them count in balances until their date.
 */
export function FutureView({ walletId }: { walletId?: string }) {
  const walletQ = useQuery(
    WalletService.method.getWallet,
    { id: walletId ?? "", today: today() },
    { enabled: Boolean(walletId) },
  );
  const wallet = walletQ.data?.wallet;
  const { byId: walletsById } = useWallets();
  const { byId: categoriesById } = useCategories();
  const { minorUnits } = useCurrencies();
  const members = useMembers();
  const [memberIds, setMemberIds] = useState<string[]>([]);
  const q = useQuery(TransactionService.method.listUpcomingTransactions, {
    walletId: walletId ?? "",
    today: today(),
    memberIds,
  });
  const transactions = q.data?.transactions ?? [];
  const [editor, setEditor] = useState<{ open: boolean; editing: Transaction | null }>({
    open: false,
    editing: null,
  });

  if (walletId && walletQ.isError) {
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

  // Group by day, keeping server order (soonest first).
  const days: { date: string; items: Transaction[] }[] = [];
  for (const t of transactions) {
    let day = days.at(-1);
    if (!day || day.date !== t.occurredOn) {
      day = { date: t.occurredOn, items: [] };
      days.push(day);
    }
    day.items.push(t);
  }

  return (
    <div className="flex flex-col gap-8">
      {walletId ? (
        <WalletHeader walletId={walletId} wallet={wallet} />
      ) : (
        <SectionHeader code="S-01" label="Wallets" title="Future" ruler />
      )}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <p className="max-w-2xl text-body-md text-ink-soft">
          Upcoming transactions, soonest first. Recurring ones are created about three months ahead.
          None of them count in balances or budgets until their date.
        </p>
        {members.shown && (
          <Popover>
            <PopoverTrigger asChild>
              <Button variant="secondary" size="sm" className="self-start">
                <Users strokeWidth={1.25} /> People
                {memberIds.length ? ` · ${memberIds.length}` : ""}
              </Button>
            </PopoverTrigger>
            <PopoverContent align="end" className="w-72">
              <PeopleFilter people={members.people} value={memberIds} onChange={setMemberIds} />
            </PopoverContent>
          </Popover>
        )}
      </div>

      {q.isPending ? (
        <Panel>
          <div className="flex flex-col gap-4">
            {[0, 1, 2].map((i) => (
              <Skeleton key={i} className="h-10" />
            ))}
          </div>
        </Panel>
      ) : transactions.length === 0 ? (
        <Panel recess notch="sm">
          <p className="text-body-md text-ink-soft">
            Nothing upcoming. Set a transaction to repeat and its next ones show up here.
          </p>
        </Panel>
      ) : (
        <Panel bodyClassName="p-0 sm:p-0" notch="md">
          <div className="flex flex-col">
            {days.map((day) => (
              <section key={day.date} aria-label={formatDay(day.date)}>
                <header className="border-b border-seam px-4 pt-5 pb-2 sm:px-6">
                  <h2 className="hud text-ink-soft">{formatDay(day.date)}</h2>
                </header>
                <ul>
                  {day.items.map((t) => {
                    const transfer =
                      t.kind === TransactionKind.TRANSFER_IN ||
                      t.kind === TransactionKind.TRANSFER_OUT;
                    const cat = categoriesById.get(t.categoryId);
                    const cp = t.counterpart;
                    const rowWallet = walletsById.get(t.walletId);
                    return (
                      <li
                        key={t.id}
                        className="border-b border-seam/60 last:border-b-0 hover:bg-shell-recess/60"
                      >
                        <button
                          type="button"
                          onClick={() => setEditor({ open: true, editing: t })}
                          className="flex w-full min-w-0 items-center gap-3 px-4 py-3 text-left outline-none focus-visible:underline sm:px-6"
                        >
                          {transfer ? (
                            <CategoryIcon icon="arrow-left-right" color="cat-slate" />
                          ) : (
                            <CategoryIcon
                              icon={cat?.icon ?? "circle-help"}
                              color={cat?.color ?? "cat-slate"}
                            />
                          )}
                          <span className="flex min-w-0 flex-1 flex-col gap-0.5 sm:flex-row sm:items-center sm:gap-4">
                            <span className="flex w-36 shrink-0 items-center gap-1.5 text-label text-ink">
                              <span className="truncate">
                                {transfer ? "Transfer" : (cat?.name ?? "Uncategorised")}
                              </span>
                              <RepeatMarker transaction={t} />
                            </span>
                            <span className="min-w-0 flex-1 truncate text-body-sm text-ink-soft">
                              {!walletId && rowWallet && (
                                <span className="text-ink">
                                  {rowWallet.name}
                                  {transfer || t.note ? " · " : ""}
                                </span>
                              )}
                              {transfer && cp && (
                                <span className="text-ink">
                                  {t.kind === TransactionKind.TRANSFER_OUT ? "To " : "From "}
                                  <strong>{cp.walletName}</strong>
                                  {t.note ? " · " : ""}
                                </span>
                              )}
                              {t.note}
                            </span>
                          </span>
                          {members.shown && members.byId.has(t.createdBy) && (
                            <MemberAvatar
                              member={members.byId.get(t.createdBy) as Member}
                              className="size-6 text-body-sm"
                            />
                          )}
                          <Amount
                            minor={signed(t)}
                            currency={t.currency}
                            minorUnits={minorUnits(t.currency)}
                            code={!walletId}
                            className="shrink-0"
                          />
                        </button>
                      </li>
                    );
                  })}
                </ul>
              </section>
            ))}
          </div>
        </Panel>
      )}

      <TransactionSheet
        open={editor.open}
        onOpenChange={(open) => setEditor((e) => ({ ...e, open }))}
        wallet={editor.editing ? walletsById.get(editor.editing.walletId) : wallet}
        editing={editor.editing}
      />
    </div>
  );
}
