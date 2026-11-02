"use client";

import type { Wallet } from "@ledger-01/api";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@ledger-01/ui/components/select";

export function WalletSelect({
  wallets,
  value,
  onChange,
  exclude,
  id,
  placeholder = "Choose a wallet",
}: {
  wallets: Wallet[];
  value: string;
  onChange: (id: string) => void;
  exclude?: string;
  id?: string;
  placeholder?: string;
}) {
  return (
    // Always controlled: "" shows the placeholder.
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger id={id} className="w-full">
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        {wallets
          .filter((w) => w.id !== exclude && (!w.archived || w.id === value))
          .map((w) => (
            <SelectItem key={w.id} value={w.id}>
              <span className="truncate">{w.name}</span>
              <span className="hud-sm text-ink-soft">{w.currency}</span>
            </SelectItem>
          ))}
      </SelectContent>
    </Select>
  );
}
