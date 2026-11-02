"use client";

import type { Category } from "@ledger-01/api";
import { CategoryIcon } from "@ledger-01/ui/components/category-icon";
import { Combobox } from "@ledger-01/ui/components/combobox";
import Link from "next/link";

/**
 * Category combobox for one kind, hiding categories hidden in (or archived from) the wallet.
 * compact shows the trigger as a small chip, for long lists where the category is already set.
 */
export function CategoryPicker({
  categories,
  walletId,
  value,
  onChange,
  id,
  compact,
}: {
  categories: Category[];
  walletId: string;
  value: string;
  onChange: (id: string) => void;
  id?: string;
  compact?: boolean;
}) {
  const options = categories.filter(
    (c) => c.id === value || (!c.archived && !c.hiddenInWalletIds.includes(walletId)),
  );
  return (
    <Combobox
      id={id}
      value={value}
      onChange={onChange}
      options={options}
      getKey={(c) => c.id}
      getSearchText={(c) => c.name}
      renderOption={(c) => (
        <>
          <CategoryIcon icon={c.icon} color={c.color} size="sm" />
          <span className="truncate">{c.name}</span>
        </>
      )}
      placeholder="Choose a category"
      className={
        compact
          ? "h-8! w-auto max-w-60 gap-2 rounded-full px-2.5 text-body-sm shadow-none"
          : undefined
      }
      searchPlaceholder="Search categories…"
      emptyText="No category found."
      footer={
        <Link
          href="/settings/categories"
          className="hud-sm block rounded-full px-2 py-1 text-signal-ink hover:bg-shell-recess"
        >
          Edit categories →
        </Link>
      }
    />
  );
}
