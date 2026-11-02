"use client";

import type { Currency } from "@ledger-01/api";
import { Combobox } from "@ledger-01/ui/components/combobox";
import { useCurrencies } from "@/lib/api/hooks";

const COMMON = ["USD", "EUR", "UYU", "ARS", "BRL", "GBP", "MXN", "CAD"];

/** Searchable ISO 4217 currency picker: common currencies first. */
export function CurrencySelect({
  id,
  value,
  onChange,
}: {
  id?: string;
  value: string;
  onChange: (code: string) => void;
}) {
  const { currencies, byCode } = useCurrencies();
  const common = COMMON.map((c) => byCode.get(c)).filter((c) => c !== undefined);
  const rest = currencies.filter((c) => !COMMON.includes(c.code));
  const option = (c: Currency) => (
    <>
      <span className="hud w-12 text-ink">{c.code}</span>
      <span className="truncate text-body-sm text-ink-soft">{c.name}</span>
    </>
  );
  return (
    <Combobox
      id={id}
      value={value}
      onChange={onChange}
      groups={[
        { heading: "Common", options: common },
        { heading: "All", options: rest },
      ]}
      getKey={(c) => c.code}
      getSearchText={(c) => `${c.code} ${c.name}`}
      renderOption={option}
      placeholder={<span className="hud">{value}</span>}
      searchPlaceholder="Search currency…"
      emptyText="No currency found."
      contentClassName="min-w-72"
    />
  );
}
