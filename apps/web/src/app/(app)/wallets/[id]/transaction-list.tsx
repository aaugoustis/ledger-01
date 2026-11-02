"use client";

import {
  type Category,
  type Label,
  type Member,
  type Transaction,
  TransactionKind,
  type Wallet,
} from "@ledger-01/api";
import { Amount } from "@ledger-01/ui/components/amount";
import { Button } from "@ledger-01/ui/components/button";
import { CategoryIcon, ColorDot } from "@ledger-01/ui/components/category-icon";
import { Checkbox } from "@ledger-01/ui/components/checkbox";
import { Panel } from "@ledger-01/ui/components/panel";
import { Skeleton } from "@ledger-01/ui/components/skeleton";
import { MemberAvatar } from "@/components/member-avatar";
import { RepeatMarker } from "@/components/repeat-marker";
import { formatDay } from "@/lib/period";

/** Signed amount from this wallet's point of view. */
export function signed(t: Transaction): bigint {
  return t.kind === TransactionKind.EXPENSE || t.kind === TransactionKind.TRANSFER_OUT
    ? -t.amountMinor
    : t.amountMinor;
}

export function TransactionList({
  wallet,
  transactions,
  loading,
  categoriesById,
  labelsById,
  peopleById,
  minorUnits,
  selected,
  onSelect,
  onOpen,
  hasMore,
  loadingMore,
  onLoadMore,
}: {
  wallet: Wallet;
  transactions: Transaction[];
  loading: boolean;
  categoriesById: Map<string, Category>;
  labelsById: Map<string, Label>;
  /** Set when the household has more than one person: rows then show who entered them. */
  peopleById?: Map<string, Member>;
  minorUnits: (code: string) => number;
  selected: Set<string>;
  onSelect: (s: Set<string>) => void;
  onOpen: (t: Transaction) => void;
  hasMore: boolean;
  loadingMore: boolean;
  onLoadMore: () => void;
}) {
  const units = minorUnits(wallet.currency);

  if (loading) {
    return (
      <Panel>
        <div className="flex flex-col gap-4">
          {[0, 1, 2, 3, 4].map((i) => (
            <Skeleton key={i} className="h-10" />
          ))}
        </div>
      </Panel>
    );
  }

  if (transactions.length === 0) {
    return (
      <Panel recess notch="sm">
        <p className="text-body-md text-ink-soft">No transactions in this period.</p>
      </Panel>
    );
  }

  // Group by day, keeping server order (newest first).
  const days: { date: string; items: Transaction[]; total: bigint }[] = [];
  for (const t of transactions) {
    let day = days.at(-1);
    if (!day || day.date !== t.occurredOn) {
      day = { date: t.occurredOn, items: [], total: 0n };
      days.push(day);
    }
    day.items.push(t);
    day.total += signed(t);
  }

  const toggle = (id: string) => {
    const next = new Set(selected);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    onSelect(next);
  };

  return (
    <Panel bodyClassName="p-0 sm:p-0" notch="md">
      <div className="flex flex-col">
        {days.map((day) => (
          <section key={day.date} aria-label={formatDay(day.date)}>
            <header className="flex items-center justify-between border-b border-seam px-4 pt-5 pb-2 sm:px-6">
              <h2 className="hud text-ink-soft">{formatDay(day.date)}</h2>
              <Amount
                minor={day.total}
                currency={wallet.currency}
                minorUnits={units}
                size="sm"
                code={false}
              />
            </header>
            <ul>
              {day.items.map((t) => {
                const transfer =
                  t.kind === TransactionKind.TRANSFER_IN || t.kind === TransactionKind.TRANSFER_OUT;
                const cat = categoriesById.get(t.categoryId);
                const cp = t.counterpart;
                return (
                  <li
                    key={t.id}
                    className="group flex items-center gap-3 border-b border-seam/60 px-4 py-3 last:border-b-0 hover:bg-shell-recess/60 sm:px-6"
                  >
                    <Checkbox
                      aria-label="Select transaction"
                      checked={selected.has(t.id)}
                      onCheckedChange={() => toggle(t.id)}
                      className="shrink-0"
                    />
                    <button
                      type="button"
                      onClick={() => onOpen(t)}
                      className="flex min-w-0 flex-1 items-center gap-3 text-left outline-none focus-visible:underline"
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
                          {transfer && cp && (
                            <span className="text-ink">
                              {t.kind === TransactionKind.TRANSFER_OUT ? "To " : "From "}
                              <strong>{cp.walletName}</strong>
                              {t.note ? " · " : ""}
                            </span>
                          )}
                          {t.note}
                        </span>
                        {t.labelIds.length > 0 && (
                          <span className="hidden shrink-0 items-center gap-2 lg:flex">
                            {t.labelIds.map((id) => {
                              const l = labelsById.get(id);
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
                        )}
                      </span>
                      {peopleById && t.createdBy && peopleById.has(t.createdBy) && (
                        <MemberAvatar
                          member={peopleById.get(t.createdBy) as Member}
                          className="size-6 text-body-sm"
                        />
                      )}
                      <span className="flex shrink-0 flex-col items-end">
                        <Amount
                          minor={signed(t)}
                          currency={wallet.currency}
                          minorUnits={units}
                          code={false}
                        />
                        {transfer && cp && cp.currency !== wallet.currency && (
                          <Amount
                            // Same direction as this leg: "−120.00 USD / −4,700.00 UYU".
                            minor={
                              t.kind === TransactionKind.TRANSFER_OUT
                                ? -cp.amountMinor
                                : cp.amountMinor
                            }
                            currency={cp.currency}
                            minorUnits={minorUnits(cp.currency)}
                            size="sm"
                            tone="muted"
                          />
                        )}
                      </span>
                    </button>
                  </li>
                );
              })}
            </ul>
          </section>
        ))}
        {hasMore && (
          <div className="flex justify-center p-4">
            <Button variant="secondary" size="sm" onClick={onLoadMore} disabled={loadingMore}>
              Load more
            </Button>
          </div>
        )}
      </div>
    </Panel>
  );
}
