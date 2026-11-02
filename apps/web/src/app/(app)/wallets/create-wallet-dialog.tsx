"use client";

import { useMutation } from "@connectrpc/connect-query";
import { WalletService } from "@ledger-01/api";
import { Button } from "@ledger-01/ui/components/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@ledger-01/ui/components/dialog";
import { Field } from "@ledger-01/ui/components/field";
import { Input } from "@ledger-01/ui/components/input";
import { MoneyInput } from "@ledger-01/ui/components/money-input";
import { parseAmount } from "@ledger-01/ui/lib/money";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { CurrencySelect } from "@/components/currency-select";
import { useCurrencies } from "@/lib/api/hooks";

export function CreateWalletDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const router = useRouter();
  const { minorUnits } = useCurrencies();
  const [name, setName] = useState("");
  const [currency, setCurrency] = useState("USD");
  const [initial, setInitial] = useState("");
  const [negative, setNegative] = useState(false);
  const create = useMutation(WalletService.method.createWallet);

  const units = minorUnits(currency);
  const amount = initial.trim() === "" ? 0n : parseAmount(initial, units);
  const valid = name.trim().length > 0 && amount !== null;

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!valid || amount === null) return;
    const res = await create.mutateAsync({
      name: name.trim(),
      currency,
      initialBalanceMinor: negative ? -amount : amount,
    });
    onOpenChange(false);
    setName("");
    setInitial("");
    if (res.wallet) router.push(`/wallets/${res.wallet.id}`);
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form onSubmit={submit} className="flex flex-col gap-6">
          <DialogHeader>
            <DialogTitle>Add wallet</DialogTitle>
            <DialogDescription>
              One wallet per account, card or cash pile. Its currency can't change later.
            </DialogDescription>
          </DialogHeader>
          <Field label="Name" htmlFor="wallet-name">
            <Input
              id="wallet-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={60}
              required
            />
          </Field>
          <Field label="Currency" htmlFor="wallet-currency">
            <CurrencySelect id="wallet-currency" value={currency} onChange={setCurrency} />
          </Field>
          <Field
            label="Initial balance (optional)"
            htmlFor="wallet-initial"
            error={amount === null ? `Use up to ${units} decimals` : undefined}
          >
            <div className="flex gap-2">
              <Button
                type="button"
                variant="secondary"
                size="icon"
                aria-label={negative ? "Negative balance" : "Positive balance"}
                aria-pressed={negative}
                onClick={() => setNegative((n) => !n)}
                className="tabular text-amount-md"
              >
                {negative ? "−" : "+"}
              </Button>
              <MoneyInput
                id="wallet-initial"
                currency={currency}
                value={initial}
                onChange={(e) => setInitial(e.target.value)}
              />
            </div>
          </Field>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={!valid || create.isPending}>
              Create wallet
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
