"use client";

import { Code, ConnectError } from "@connectrpc/connect";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import {
  type Budget,
  BudgetRepeat,
  BudgetService,
  type Category,
  CategoryKind,
  UserService,
  type Wallet,
} from "@ledger-01/api";
import { Button } from "@ledger-01/ui/components/button";
import { CategoryIcon } from "@ledger-01/ui/components/category-icon";
import { DatePicker } from "@ledger-01/ui/components/date-picker";
import { Field } from "@ledger-01/ui/components/field";
import { Input } from "@ledger-01/ui/components/input";
import { MoneyInput } from "@ledger-01/ui/components/money-input";
import { MultiCombobox } from "@ledger-01/ui/components/multi-combobox";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@ledger-01/ui/components/select";
import { fault } from "@ledger-01/ui/lib/fault";
import { formatAmountInput, parseAmount } from "@ledger-01/ui/lib/money";
import { useId, useState } from "react";
import { toast } from "sonner";
import { CurrencySelect } from "@/components/currency-select";
import { repeatOptions } from "@/lib/budgets";
import { currentMonth } from "@/lib/period";

type FieldKey = "name" | "amount" | "currency" | "repeat" | "start" | "end" | "wallets" | "cats";

const serverFields: Record<string, FieldKey> = {
  name: "name",
  amount_minor: "amount",
  currency: "currency",
  repeat: "repeat",
  start_date: "start",
  end_date: "end",
  wallet_ids: "wallets",
  category_ids: "cats",
};

/** Create or edit a budget. Fields are in fill order, so the form flows top to bottom. */
export function BudgetForm({
  editing,
  wallets,
  categories,
  minorUnits,
  onDone,
}: {
  editing: Budget | null;
  wallets: Wallet[];
  categories: Category[];
  minorUnits: (code: string) => number;
  onDone: (budget: Budget) => void;
}) {
  const id = useId();
  const { data: me } = useQuery(UserService.method.getMe, {});
  const [name, setName] = useState(editing?.name ?? "");
  const [currency, setCurrency] = useState(
    editing?.currency ?? me?.household?.baseCurrency ?? "USD",
  );
  const [amount, setAmount] = useState(
    editing ? formatAmountInput(editing.amountMinor, minorUnits(editing.currency)) : "",
  );
  const [repeat, setRepeat] = useState<BudgetRepeat>(editing?.repeat ?? BudgetRepeat.MONTHLY);
  const [start, setStart] = useState(editing?.startDate ?? currentMonth().from);
  const [end, setEnd] = useState(editing?.endDate || currentMonth().to);
  const [walletIds, setWalletIds] = useState<string[]>(editing?.walletIds ?? []);
  const [categoryIds, setCategoryIds] = useState<string[]>(editing?.categoryIds ?? []);
  const create = useMutation(BudgetService.method.createBudget);
  const update = useMutation(BudgetService.method.updateBudget);

  const units = minorUnits(currency);
  const parsedAmount = amount.trim() === "" ? null : parseAmount(amount, units);
  const once = repeat === BudgetRepeat.ONCE;

  const [errors, setErrors] = useState<Partial<Record<FieldKey, string>>>({});
  const clear = (key: FieldKey) => setErrors((e) => (e[key] ? { ...e, [key]: undefined } : e));
  const fieldId = (key: FieldKey) => `${id}-${key}`;
  function reject(key: FieldKey, message: string, opts: { toast?: boolean } = {}) {
    setErrors({ [key]: message });
    if (opts.toast !== false) toast.error(message);
    fault(fieldId(key));
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setErrors({});
    if (!name.trim()) return reject("name", "Name the budget");
    if (parsedAmount === null || parsedAmount <= 0n) return reject("amount", "Enter an amount");
    if (!start) return reject("start", "Pick a start date");
    if (once && !end) return reject("end", "Pick an end date");
    if (once && end < start) return reject("end", "End the budget on or after its start");
    const input = {
      name: name.trim(),
      amountMinor: parsedAmount,
      currency,
      repeat,
      startDate: start,
      endDate: once ? end : "",
      walletIds,
      categoryIds,
    };
    try {
      const res = editing
        ? await update.mutateAsync({ id: editing.id, budget: input })
        : await create.mutateAsync({ budget: input });
      toast.success(editing ? "Saved" : "Budget added");
      if (res.budget) onDone(res.budget);
    } catch (err) {
      // The API names the offending field ("end_date: must not be before…"); point at it.
      if (!(err instanceof ConnectError) || err.code !== Code.InvalidArgument) return;
      const m = /^([a-z_.]+): (.+)$/s.exec(err.rawMessage);
      const key = m?.[1] ? serverFields[m[1]] : undefined;
      if (key && m?.[2]) reject(key, m[2], { toast: false });
    }
  }

  const expenseCategories = categories.filter(
    (c) => c.kind === CategoryKind.EXPENSE && (!c.archived || categoryIds.includes(c.id)),
  );
  const activeWallets = wallets.filter((w) => !w.archived || walletIds.includes(w.id));

  return (
    <form onSubmit={submit} className="flex flex-col gap-5" noValidate>
      <Field label="Name" htmlFor={fieldId("name")} error={errors.name}>
        <Input
          id={fieldId("name")}
          value={name}
          onChange={(e) => {
            setName(e.target.value);
            clear("name");
          }}
          maxLength={60}
          placeholder="Groceries, Eating out, Trip…"
        />
      </Field>

      <div className="grid gap-4 sm:grid-cols-[1fr_160px]">
        <Field
          label="Amount"
          htmlFor={fieldId("amount")}
          error={
            errors.amount ??
            (amount !== "" && parsedAmount === null ? `Use up to ${units} decimals` : undefined)
          }
        >
          <MoneyInput
            id={fieldId("amount")}
            currency={currency}
            value={amount}
            onChange={(e) => {
              setAmount(e.target.value);
              clear("amount");
            }}
            aria-invalid={amount !== "" && parsedAmount === null}
          />
        </Field>
        <Field label="Currency" htmlFor={fieldId("currency")} error={errors.currency}>
          <CurrencySelect
            id={fieldId("currency")}
            value={currency}
            onChange={(c) => {
              setCurrency(c);
              clear("currency");
            }}
          />
        </Field>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Repeats" htmlFor={fieldId("repeat")} error={errors.repeat}>
          <Select
            value={String(repeat)}
            onValueChange={(v) => {
              setRepeat(Number(v) as BudgetRepeat);
              clear("repeat");
            }}
          >
            <SelectTrigger id={fieldId("repeat")} className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {repeatOptions.map((o) => (
                <SelectItem key={o.value} value={String(o.value)}>
                  {o.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>
        <Field
          label={once ? "Starts" : "First period starts"}
          htmlFor={fieldId("start")}
          error={errors.start}
        >
          <DatePicker
            id={fieldId("start")}
            value={start}
            onChange={(v) => {
              setStart(v);
              clear("start");
            }}
          />
        </Field>
      </div>

      {once && (
        <Field label="Ends" htmlFor={fieldId("end")} error={errors.end}>
          <DatePicker
            id={fieldId("end")}
            value={end}
            onChange={(v) => {
              setEnd(v);
              clear("end");
            }}
          />
        </Field>
      )}

      <Field
        label="Wallets"
        htmlFor={fieldId("wallets")}
        error={errors.wallets}
        help="Expenses from these wallets count. Leave it on all to count every wallet."
      >
        <MultiCombobox
          id={fieldId("wallets")}
          value={walletIds}
          onChange={(v) => {
            setWalletIds(v);
            clear("wallets");
          }}
          options={activeWallets}
          getKey={(w) => w.id}
          getSearchText={(w) => `${w.name} ${w.currency}`}
          renderOption={(w) => (
            <>
              <span className="truncate">{w.name}</span>
              <span className="hud-sm text-ink-soft">{w.currency}</span>
            </>
          )}
          allLabel="All wallets"
          noun="wallets"
          searchPlaceholder="Search wallets…"
        />
      </Field>

      <Field
        label="Categories"
        htmlFor={fieldId("cats")}
        error={errors.cats}
        help="Only expenses count. Leave it on all to count every expense category."
      >
        <MultiCombobox
          id={fieldId("cats")}
          value={categoryIds}
          onChange={(v) => {
            setCategoryIds(v);
            clear("cats");
          }}
          options={expenseCategories}
          getKey={(c) => c.id}
          getSearchText={(c) => c.name}
          renderOption={(c) => (
            <>
              <CategoryIcon icon={c.icon} color={c.color} size="sm" />
              <span className="truncate">{c.name}</span>
            </>
          )}
          allLabel="All expense categories"
          noun="categories"
          searchPlaceholder="Search categories…"
        />
      </Field>

      <div className="flex justify-end gap-2 pt-2">
        <Button type="submit" disabled={create.isPending || update.isPending}>
          {editing ? "Save budget" : "Add budget"}
        </Button>
      </div>
    </form>
  );
}
