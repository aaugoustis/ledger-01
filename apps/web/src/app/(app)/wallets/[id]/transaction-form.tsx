"use client";

import { Code, ConnectError } from "@connectrpc/connect";
import { useMutation } from "@connectrpc/connect-query";
import {
  type Category,
  CategoryKind,
  type Label,
  OccurrenceScope,
  RecurringFrequency,
  type Transaction,
  TransactionKind,
  TransactionService,
  type Wallet,
} from "@ledger-01/api";
import { Button } from "@ledger-01/ui/components/button";
import { Checkbox } from "@ledger-01/ui/components/checkbox";
import { DatePicker } from "@ledger-01/ui/components/date-picker";
import { Field, FieldError } from "@ledger-01/ui/components/field";
import { FxConverter } from "@ledger-01/ui/components/fx-converter";
import { Input } from "@ledger-01/ui/components/input";
import { Label as UILabel } from "@ledger-01/ui/components/label";
import { MoneyInput } from "@ledger-01/ui/components/money-input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@ledger-01/ui/components/select";
import { Tabs, TabsList, TabsTrigger } from "@ledger-01/ui/components/tabs";
import { fault } from "@ledger-01/ui/lib/fault";
import { editFx, type FxState, fxResult, initialFxState } from "@ledger-01/ui/lib/fx";
import { formatAmountInput, parseAmount } from "@ledger-01/ui/lib/money";
import { Trash2 } from "lucide-react";
import { useId, useState } from "react";
import { toast } from "sonner";
import { CategoryPicker } from "@/components/category-picker";
import { LabelPicker } from "@/components/label-picker";
import { ScopeDialog } from "@/components/scope-dialog";
import { WalletSelect } from "@/components/wallet-select";
import { rememberWallet } from "@/lib/last-wallet";
import { today } from "@/lib/period";
import {
  endOptions,
  frequencyOptions,
  type RepeatEnd,
  type RepeatState,
  repeatChanged,
  repeatInput,
  repeatState,
} from "@/lib/recurrence";

type Mode = "expense" | "income" | "transfer";

/** Form controls that can show an error; each control's id is `${formId}-${key}`. */
type FieldKey =
  | "cat"
  | "amount"
  | "tamount"
  | "fx"
  | "from"
  | "to"
  | "date"
  | "note"
  | "labels"
  | "repeat"
  | "end"
  | "endDate"
  | "count";

/** Maps an API field name to the control that edits it. */
function serverField(field: string, crossCurrency: boolean): FieldKey | undefined {
  switch (field) {
    case "category_id":
      return "cat";
    case "amount_minor":
      return "amount";
    case "rate":
    case "from_amount_minor":
    case "to_amount_minor":
      return crossCurrency ? "fx" : "tamount";
    case "from_wallet_id":
      return "from";
    case "to_wallet_id":
    case "wallet":
      return "to";
    case "occurred_on":
      return "date";
    case "note":
      return "note";
    case "label_ids":
      return "labels";
    case "repeat":
    case "repeat.frequency":
      return "repeat";
    case "repeat.end_date":
      return "endDate";
    case "repeat.end_count":
      return "count";
  }
  return undefined;
}

function modeOf(t: Transaction): Mode {
  if (t.kind === TransactionKind.INCOME) return "income";
  if (t.kind === TransactionKind.EXPENSE) return "expense";
  return "transfer";
}

/** A save or delete of a recurring row waiting for its scope. */
type Pending =
  | {
      kind: "save";
      run: (scope: OccurrenceScope) => Promise<void>;
      /** The repeat changed, so only "this and future ones" applies. */
      series: boolean;
      /** The repeat changed to never: this occurrence becomes the last. */
      stops: boolean;
    }
  | { kind: "delete" };

/**
 * New/edit form for expenses, income and transfers, one tab each. Cross-currency
 * transfers use the three-field converter: sent, received and rate. Any of them can repeat; saving
 * or deleting an occurrence of a recurring one asks whether it applies to only that occurrence or
 * to the ones after it too.
 *
 * With onWalletChange (a new transaction started outside a wallet page), expenses and income show
 * a Wallet field after the amount, and a transfer's From wallet follows it.
 */
export function TransactionForm({
  wallet,
  wallets,
  categories,
  labels,
  minorUnits,
  editing,
  onDone,
  onWalletChange,
}: {
  wallet: Wallet;
  wallets: Wallet[];
  categories: Category[];
  labels: Label[];
  minorUnits: (code: string) => number;
  editing?: Transaction | null;
  onDone: () => void;
  onWalletChange?: (id: string) => void;
}) {
  const id = useId();
  const isEdit = Boolean(editing);
  const [mode, setMode] = useState<Mode>(editing ? modeOf(editing) : "expense");
  const [categoryId, setCategoryId] = useState(editing?.categoryId ?? "");
  const [date, setDate] = useState(editing?.occurredOn ?? today());
  const [note, setNote] = useState(editing?.note ?? "");
  const [labelIds, setLabelIds] = useState<string[]>(editing?.labelIds ?? []);
  const [amount, setAmount] = useState(
    editing && modeOf(editing) !== "transfer"
      ? formatAmountInput(editing.amountMinor, minorUnits(wallet.currency))
      : "",
  );
  const [keepOpen, setKeepOpen] = useState(false);
  const recurring = Boolean(editing?.recurringRuleId);
  const [repeat, setRepeat] = useState<RepeatState>(() => repeatState(editing?.recurrence));
  const repeats = repeat.frequency !== RecurringFrequency.UNSPECIFIED;
  const [pendingScope, setPendingScope] = useState<Pending | null>(null);

  // Transfer state. Editing a leg reconstructs the transfer from its counterpart.
  const leg = editing?.counterpart;
  const legIsOut = editing?.kind === TransactionKind.TRANSFER_OUT;
  const [fromId, setFromId] = useState(leg ? (legIsOut ? wallet.id : leg.walletId) : wallet.id);
  const [toId, setToId] = useState(leg ? (legIsOut ? leg.walletId : wallet.id) : "");
  const from = wallets.find((w) => w.id === fromId);
  const to = wallets.find((w) => w.id === toId);
  const fromUnits = minorUnits(from?.currency ?? wallet.currency);
  const toUnits = minorUnits(to?.currency ?? wallet.currency);
  const crossCurrency = Boolean(from && to && from.currency !== to.currency);
  const [fx, setFx] = useState<FxState>(() => {
    if (!editing || !leg) return initialFxState();
    const fromMinor = legIsOut ? editing.amountMinor : leg.amountMinor;
    const toMinor = legIsOut ? leg.amountMinor : editing.amountMinor;
    let s = initialFxState();
    const c = { fromMinorUnits: fromUnits, toMinorUnits: toUnits };
    s = editFx(s, "from", formatAmountInput(fromMinor, fromUnits), c);
    s = editFx(s, "to", formatAmountInput(toMinor, toUnits), c);
    return s;
  });
  const [transferAmount, setTransferAmount] = useState(
    editing && leg ? formatAmountInput(editing.amountMinor, minorUnits(wallet.currency)) : "",
  );

  const create = useMutation(TransactionService.method.createTransaction);
  const update = useMutation(TransactionService.method.updateTransaction);
  const createTransfer = useMutation(TransactionService.method.createTransfer);
  const updateTransfer = useMutation(TransactionService.method.updateTransfer);
  const remove = useMutation(TransactionService.method.deleteTransactions);
  const pending =
    create.isPending ||
    update.isPending ||
    createTransfer.isPending ||
    updateTransfer.isPending ||
    remove.isPending;

  const kindCategories = categories.filter(
    (c) => c.kind === (mode === "income" ? CategoryKind.INCOME : CategoryKind.EXPENSE),
  );
  const parsedAmount = parseAmount(amount, minorUnits(wallet.currency));

  function changeMode(next: string) {
    setMode(next as Mode);
    setCategoryId("");
    setErrors({});
  }

  function changeWallets(nextFrom: string, nextTo: string) {
    setFromId(nextFrom);
    setToId(nextTo);
    setFx(initialFxState());
    if (nextFrom !== fromId) onWalletChange?.(nextFrom);
  }

  function resetForNext() {
    setAmount("");
    setNote("");
    setLabelIds([]);
    setTransferAmount("");
    setFx(initialFxState());
    setRepeat(repeatState());
  }

  function changeRepeat(next: Partial<RepeatState>) {
    setRepeat((r) => ({ ...r, ...next }));
    for (const key of ["repeat", "endDate", "count"] as const) clear(key);
  }

  // --- Field errors. A rejected field jitters and pings (fault() in the UI kit), takes focus and
  // shows a HUD fault line; the toast still reports it too. Editing the field clears its error.
  const [errors, setErrors] = useState<Partial<Record<FieldKey, string>>>({});
  const clear = (key: FieldKey) => setErrors((e) => (e[key] ? { ...e, [key]: undefined } : e));
  const fieldId = (key: FieldKey) => `${id}-${key}`;

  function reject(key: FieldKey, message: string, opts: { toast?: boolean } = {}) {
    setErrors({ [key]: message });
    if (opts.toast !== false) toast.error(message);
    fault(fieldId(key));
  }

  // The API names the offending field ("rate: amounts don't match…"); point at it.
  function rejectFromServer(err: unknown) {
    if (!(err instanceof ConnectError) || err.code !== Code.InvalidArgument) return;
    const m = /^([a-z_.]+): (.+)$/s.exec(err.rawMessage);
    const key = m?.[1] ? serverField(m[1], crossCurrency) : undefined;
    if (key && m?.[2]) reject(key, m[2], { toast: false }); // the mutation already toasted
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setErrors({});
    const run = validate();
    if (!run) return;
    if (!recurring) return finish(run, OccurrenceScope.UNSPECIFIED);
    // A new schedule only makes sense for the series; otherwise ask.
    const original = repeatInput(repeatState(editing?.recurrence), editing?.occurredOn ?? date);
    const rep = repeatInput(repeat, date);
    const series = original.ok && rep.ok && repeatChanged(original.value, rep.value);
    const stops = series && repeat.frequency === RecurringFrequency.UNSPECIFIED;
    setPendingScope({ kind: "save", run, series, stops });
  }

  async function finish(run: (scope: OccurrenceScope) => Promise<void>, scope: OccurrenceScope) {
    try {
      await run(scope);
    } catch (err) {
      rejectFromServer(err);
      return;
    }
    toast.success(isEdit ? "Saved" : "Added");
    // The next transaction started outside a wallet page starts here.
    if (!isEdit && mode !== "transfer") rememberWallet(wallet.id);
    if (!isEdit && keepOpen) resetForNext();
    else onDone();
  }

  async function deleteEditing(scope: OccurrenceScope) {
    if (!editing) return;
    try {
      await remove.mutateAsync({ ids: [editing.id], scope, today: today() });
    } catch {
      return; // the mutation already toasted
    }
    toast.success("Deleted");
    onDone();
  }

  /**
   * Checks the fields, faulting the first bad one, and returns the save to run with a scope (or
   * nothing when a field is wrong).
   */
  function validate(): ((scope: OccurrenceScope) => Promise<void>) | undefined {
    const rep = repeatInput(repeat, date);
    if (mode === "transfer") {
      if (!from) return void reject("from", "Choose the wallet to transfer from");
      if (!to) return void reject("to", "Choose the wallet to transfer to");
      let fromMinor: bigint;
      let toMinor: bigint;
      let rate = "1";
      if (crossCurrency) {
        const r = fxResult(fx, { fromMinorUnits: fromUnits, toMinorUnits: toUnits });
        if (!r) return void reject("fx", "Fill in two of: sent, received, rate");
        ({ fromMinor, toMinor, rate } = r);
      } else {
        const a = parseAmount(transferAmount, fromUnits);
        if (a === null || a <= 0n) return void reject("tamount", "Enter an amount");
        fromMinor = toMinor = a;
      }
      if (!rep.ok) return void reject(rep.field, rep.message);
      const body = {
        fromWalletId: from.id,
        toWalletId: to.id,
        fromAmountMinor: fromMinor,
        toAmountMinor: toMinor,
        rate,
        occurredOn: date,
        note,
        labelIds,
        repeat: rep.value,
        today: today(),
      };
      return async (scope) => {
        if (isEdit && leg) await updateTransfer.mutateAsync({ id: leg.transferId, scope, ...body });
        else await createTransfer.mutateAsync(body);
      };
    }
    if (!categoryId) return void reject("cat", "Choose a category");
    if (parsedAmount === null || parsedAmount <= 0n)
      return void reject("amount", "Enter an amount");
    if (!rep.ok) return void reject(rep.field, rep.message);
    const body = {
      categoryId,
      amountMinor: parsedAmount,
      occurredOn: date,
      note,
      labelIds,
      repeat: rep.value,
      today: today(),
    };
    return async (scope) => {
      if (isEdit && editing) await update.mutateAsync({ id: editing.id, scope, ...body });
      else
        await create.mutateAsync({
          walletId: wallet.id,
          kind: mode === "income" ? TransactionKind.INCOME : TransactionKind.EXPENSE,
          ...body,
        });
    };
  }

  const isTransferEdit = isEdit && mode === "transfer";

  return (
    <form onSubmit={submit} className="flex flex-col gap-5">
      <Tabs value={mode} onValueChange={changeMode}>
        <TabsList className="w-full">
          <TabsTrigger value="expense" disabled={isTransferEdit}>
            Expense
          </TabsTrigger>
          <TabsTrigger value="income" disabled={isTransferEdit}>
            Income
          </TabsTrigger>
          <TabsTrigger value="transfer" disabled={isEdit && !isTransferEdit}>
            Transfer
          </TabsTrigger>
        </TabsList>
      </Tabs>

      {mode === "transfer" ? (
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="From wallet" htmlFor={`${id}-from`} error={errors.from}>
            <WalletSelect
              id={`${id}-from`}
              wallets={wallets}
              value={fromId}
              onChange={(v) => {
                changeWallets(v, v === toId ? "" : toId);
                clear("from");
              }}
            />
          </Field>
          <Field label="To wallet" htmlFor={`${id}-to`} error={errors.to}>
            <WalletSelect
              id={`${id}-to`}
              wallets={wallets}
              value={toId}
              exclude={fromId}
              onChange={(v) => {
                changeWallets(fromId, v);
                clear("to");
              }}
            />
          </Field>
          {crossCurrency && from && to ? (
            <div className="flex flex-col gap-1.5 sm:col-span-2">
              <FxConverter
                id={`${id}-fx`}
                invalid={Boolean(errors.fx)}
                from={{ code: from.currency, minorUnits: fromUnits }}
                to={{ code: to.currency, minorUnits: toUnits }}
                value={fx}
                onChange={(next) => {
                  setFx(next);
                  clear("fx");
                }}
              />
              <FieldError>{errors.fx}</FieldError>
            </div>
          ) : (
            <Field
              label="Amount"
              htmlFor={`${id}-tamount`}
              className="sm:col-span-2"
              error={errors.tamount}
            >
              <MoneyInput
                id={`${id}-tamount`}
                currency={from?.currency ?? wallet.currency}
                value={transferAmount}
                onChange={(e) => {
                  setTransferAmount(e.target.value);
                  clear("tamount");
                }}
              />
            </Field>
          )}
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Category" htmlFor={`${id}-cat`} error={errors.cat}>
            <CategoryPicker
              id={`${id}-cat`}
              categories={kindCategories}
              walletId={wallet.id}
              value={categoryId}
              onChange={(v) => {
                setCategoryId(v);
                clear("cat");
              }}
            />
          </Field>
          <Field
            label="Amount"
            htmlFor={`${id}-amount`}
            error={
              errors.amount ??
              (amount && parsedAmount === null
                ? `Use up to ${minorUnits(wallet.currency)} decimals`
                : undefined)
            }
          >
            <MoneyInput
              id={`${id}-amount`}
              currency={wallet.currency}
              value={amount}
              onChange={(e) => {
                setAmount(e.target.value);
                clear("amount");
              }}
              aria-invalid={amount !== "" && parsedAmount === null}
            />
          </Field>
          {onWalletChange && (
            <Field label="Wallet" htmlFor={`${id}-wallet`} className="sm:col-span-2">
              <WalletSelect
                id={`${id}-wallet`}
                wallets={wallets}
                value={wallet.id}
                onChange={(v) => changeWallets(v, v === toId ? "" : toId)}
              />
            </Field>
          )}
        </div>
      )}

      <div className="grid gap-4 sm:grid-cols-[240px_1fr]">
        <Field label="Date" htmlFor={`${id}-date`} error={errors.date}>
          <DatePicker
            id={`${id}-date`}
            value={date}
            onChange={(v) => {
              setDate(v);
              clear("date");
            }}
          />
        </Field>
        <Field label="Note (optional)" htmlFor={`${id}-note`} error={errors.note}>
          <Input
            id={`${id}-note`}
            value={note}
            onChange={(e) => {
              setNote(e.target.value);
              clear("note");
            }}
            maxLength={500}
            placeholder="Write a note"
          />
        </Field>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Repeat" htmlFor={fieldId("repeat")} error={errors.repeat}>
          <Select
            value={String(repeat.frequency)}
            onValueChange={(v) => changeRepeat({ frequency: Number(v) as RecurringFrequency })}
          >
            <SelectTrigger id={fieldId("repeat")} className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {frequencyOptions.map((o) => (
                <SelectItem key={o.value} value={String(o.value)}>
                  {o.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>
        {repeats && (
          <Field label="Ends" htmlFor={fieldId("end")}>
            <Select value={repeat.end} onValueChange={(v) => changeRepeat({ end: v as RepeatEnd })}>
              <SelectTrigger id={fieldId("end")} className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {endOptions.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
        )}
        {repeats && repeat.end === "date" && (
          <Field label="Last date" htmlFor={fieldId("endDate")} error={errors.endDate}>
            <DatePicker
              id={fieldId("endDate")}
              value={repeat.endDate}
              onChange={(v) => changeRepeat({ endDate: v })}
            />
          </Field>
        )}
        {repeats && repeat.end === "count" && (
          <Field label="Times" htmlFor={fieldId("count")} error={errors.count}>
            <Input
              id={fieldId("count")}
              inputMode="numeric"
              value={repeat.count}
              onChange={(e) => changeRepeat({ count: e.target.value })}
              placeholder="12"
            />
          </Field>
        )}
      </div>

      <Field label="Labels (optional)" htmlFor={`${id}-labels`} error={errors.labels}>
        <LabelPicker
          id={`${id}-labels`}
          labels={labels}
          value={labelIds}
          onChange={(v) => {
            setLabelIds(v);
            clear("labels");
          }}
        />
      </Field>

      <div className="flex flex-wrap items-center justify-between gap-4 pt-2">
        {!isEdit ? (
          <div className="flex items-center gap-2">
            <Checkbox
              id={`${id}-keep`}
              checked={keepOpen}
              onCheckedChange={(v) => setKeepOpen(v === true)}
            />
            <UILabel htmlFor={`${id}-keep`} className="text-body-sm text-ink-soft">
              Keep open to add more
            </UILabel>
          </div>
        ) : (
          <Button
            type="button"
            variant="ghost"
            onClick={() => setPendingScope({ kind: "delete" })}
            disabled={pending}
          >
            <Trash2 strokeWidth={1.25} /> Delete
          </Button>
        )}
        <div className="flex gap-2">
          <Button type="button" variant="ghost" onClick={onDone}>
            Cancel
          </Button>
          <Button type="submit" disabled={pending}>
            {isEdit ? "Save changes" : mode === "transfer" ? "Add transfer" : "Add transaction"}
          </Button>
        </div>
      </div>

      {pendingScope?.kind === "save" && (
        <ScopeDialog
          open
          onOpenChange={(open) => !open && setPendingScope(null)}
          title="Save a recurring transaction"
          description={
            pendingScope.stops
              ? "This one becomes the last: the upcoming ones after it are deleted. Past ones stay as they are."
              : pendingScope.series
                ? "The new repeat applies to this one and the ones after it. Past ones stay as they are."
                : "Change only this one, or this one and the ones after it? Past ones stay as they are."
          }
          offerOnly={!pendingScope.series}
          disabled={pending}
          onChoose={(scope) => {
            const { run } = pendingScope;
            setPendingScope(null);
            void finish(run, scope);
          }}
        />
      )}
      {pendingScope?.kind === "delete" && (
        <ScopeDialog
          open
          onOpenChange={(open) => !open && setPendingScope(null)}
          title={recurring ? "Delete a recurring transaction" : "Delete this transaction?"}
          description={
            recurring
              ? "Delete only this one, or this one and the ones after it? Past ones stay."
              : mode === "transfer"
                ? "The transfer is deleted from both wallets. This can't be undone."
                : "This can't be undone."
          }
          offerOnly={recurring}
          future={recurring ? "This and future ones" : "Delete"}
          destructive
          disabled={pending}
          onChoose={(scope) => {
            setPendingScope(null);
            void deleteEditing(recurring ? scope : OccurrenceScope.UNSPECIFIED);
          }}
        />
      )}
    </form>
  );
}
