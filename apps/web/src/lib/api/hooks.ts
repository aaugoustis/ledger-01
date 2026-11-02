"use client";

import { useQuery } from "@connectrpc/connect-query";
import {
  type Category,
  CategoryService,
  HouseholdService,
  type Label,
  LabelService,
  type Member,
  UserService,
  type Wallet,
  WalletService,
} from "@ledger-01/api";
import { useMemo } from "react";
import { today } from "@/lib/period";

export function useCurrencies() {
  const q = useQuery(
    UserService.method.listCurrencies,
    {},
    { staleTime: Number.POSITIVE_INFINITY },
  );
  const byCode = useMemo(
    () => new Map((q.data?.currencies ?? []).map((c) => [c.code, c])),
    [q.data],
  );
  /** Decimal places for a currency (2 until currencies load). */
  const minorUnits = (code: string) => byCode.get(code)?.minorUnits ?? 2;
  return { ...q, currencies: q.data?.currencies ?? [], byCode, minorUnits };
}

/** The household's base (reporting) currency; "USD" until it loads. */
export function useBaseCurrency() {
  return useHousehold().household?.baseCurrency || "USD";
}

/** The signed-in user's active household and their role in it. */
export function useHousehold() {
  const q = useQuery(UserService.method.getMe, {}, { staleTime: 60_000 });
  const household = q.data?.household;
  return { ...q, household, isOwner: household?.role === "owner" };
}

/**
 * Everyone who can have entered rows here: members, then former members. `shown` is false while
 * the household has only ever had one person, so the by-person filter and creator marks stay out
 * of the way.
 */
export function useMembers() {
  const q = useQuery(HouseholdService.method.listMembers, {}, { staleTime: 60_000 });
  const people: Member[] = useMemo(
    () => [...(q.data?.members ?? []), ...(q.data?.formerMembers ?? [])],
    [q.data],
  );
  const byId = useMemo(() => new Map(people.map((m) => [m.userId, m])), [people]);
  return { ...q, people, byId, shown: people.length > 1 };
}

export function useWallets(opts: { includeHidden?: boolean } = {}) {
  // Balances as of the visitor's local today: upcoming rows don't count yet.
  const q = useQuery(WalletService.method.listWallets, {
    includeHidden: opts.includeHidden ?? true,
    today: today(),
  });
  const wallets: Wallet[] = q.data?.wallets ?? [];
  const byId = useMemo(() => new Map(wallets.map((w) => [w.id, w])), [wallets]);
  return { ...q, wallets, byId };
}

export function useCategories() {
  const q = useQuery(CategoryService.method.listCategories, { includeArchived: true });
  const categories: Category[] = q.data?.categories ?? [];
  const byId = useMemo(() => new Map(categories.map((c) => [c.id, c])), [categories]);
  return { ...q, categories, byId };
}

export function useLabels() {
  const q = useQuery(LabelService.method.listLabels, {});
  const labels: Label[] = q.data?.labels ?? [];
  const byId = useMemo(() => new Map(labels.map((l) => [l.id, l])), [labels]);
  return { ...q, labels, byId };
}
