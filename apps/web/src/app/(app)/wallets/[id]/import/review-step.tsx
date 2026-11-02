"use client";

import { Code, ConnectError } from "@connectrpc/connect";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import {
  CategoryKind,
  type ImportMapping,
  type ImportRow,
  ImportService,
  TransactionKind,
  type Wallet,
} from "@ledger-01/api";
import { Amount } from "@ledger-01/ui/components/amount";
import { Badge } from "@ledger-01/ui/components/badge";
import { Button } from "@ledger-01/ui/components/button";
import { Checkbox } from "@ledger-01/ui/components/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@ledger-01/ui/components/dialog";
import { Field, FieldError } from "@ledger-01/ui/components/field";
import { Input } from "@ledger-01/ui/components/input";
import { Label } from "@ledger-01/ui/components/label";
import { Panel } from "@ledger-01/ui/components/panel";
import { Skeleton } from "@ledger-01/ui/components/skeleton";
import { ToggleGroup, ToggleGroupItem } from "@ledger-01/ui/components/toggle-group";
import { fault } from "@ledger-01/ui/lib/fault";
import { formatMoney } from "@ledger-01/ui/lib/money";
import { WandSparkles } from "lucide-react";
import Link from "next/link";
import { useEffect, useId, useState } from "react";
import { toast } from "sonner";
import { CategoryPicker } from "@/components/category-picker";
import { useCategories, useCurrencies } from "@/lib/api/hooks";
import {
  applyRule,
  canChoose,
  firstMissingCategory,
  initialCategories,
  initialSelection,
  leftOutBadge,
  needsCategory,
  type ReviewView,
  reviewCounts,
  rowRelation,
  settledGroups,
  viewCounts,
  viewRows,
} from "@/lib/imports";
import { formatDay } from "@/lib/period";

const PAGE = 50;

const viewLabels: Record<ReviewView, string> = {
  all: "All",
  ticked: "Ticked",
  uncategorised: "Needs a category",
  duplicates: "Duplicates",
  "left-out": "Left out",
};

/**
 * Every row of the file read with the mapping: tick the ones to import and give each a category.
 * Duplicates (imported before, already in the wallet, or repeated in the file) and purchases
 * refunded in full (with their refund) start unticked and say what they match; discounts merged
 * into their purchase are folded into it, and rows that can't be read can't be ticked. A discount
 * for an earlier purchase lowers it and needs no category.
 *
 * Long statements stay short: chips filter the list, the "All" view folds settled rows into one
 * line per kind, rows with a category show it as a chip, and the Import bar stays pinned.
 */
export function ReviewStep({
  batchId,
  mapping,
  wallet,
  savedMapping,
  onBack,
  onDone,
}: {
  batchId: string;
  mapping: ImportMapping;
  wallet: Wallet;
  savedMapping: boolean;
  onBack: () => void;
  onDone: (imported: bigint) => void;
}) {
  const id = useId();
  const { minorUnits } = useCurrencies();
  const { categories } = useCategories();
  const q = useQuery(
    ImportService.method.previewImport,
    { batchId, mapping, limit: 0 },
    { staleTime: Number.POSITIVE_INFINITY, retry: false },
  );
  const rows = q.data?.rows ?? [];
  const [selected, setSelected] = useState<Set<number> | null>(null);
  const [picks, setPicks] = useState<Map<number, string>>(new Map());
  const [shown, setShown] = useState(PAGE);
  const [view, setView] = useState<ReviewView>("all");
  const [showSettled, setShowSettled] = useState(false);
  const [remember, setRemember] = useState(true);
  const [missing, setMissing] = useState<number | null>(null);
  const [ruleFor, setRuleFor] = useState<ImportRow | null>(null);
  const commit = useMutation(ImportService.method.commitImport);

  // Start from the server's suggestions once the rows arrive.
  useEffect(() => {
    if (q.data && selected === null) {
      setSelected(initialSelection(q.data.rows));
      setPicks(initialCategories(q.data.rows));
    }
  }, [q.data, selected]);

  const sel = selected ?? new Set<number>();
  const counts = reviewCounts(rows, sel);
  const listed = viewRows(rows, view, sel, picks, showSettled);
  const chips = viewCounts(rows, sel, picks);
  const folded = view === "all" && !showSettled ? settledGroups(rows) : [];
  // Tick all / Untick all act on the rows the current view lists.
  const choosable = listed.filter(canChoose);
  const units = minorUnits(wallet.currency);
  const money = (minor: bigint) => formatMoney(minor, wallet.currency, units, { code: false });
  const allTicked = choosable.length > 0 && choosable.every((r) => sel.has(r.index));
  const noneTicked = choosable.every((r) => !sel.has(r.index));

  function show(next: ReviewView) {
    setView(next);
    setShown(PAGE);
  }

  function setTicks(on: boolean) {
    const next = new Set(sel);
    for (const r of choosable) {
      if (on) next.add(r.index);
      else next.delete(r.index);
    }
    setSelected(next);
  }
  const kindCategories = (kind: TransactionKind) =>
    categories.filter(
      (c) =>
        c.kind === (kind === TransactionKind.INCOME ? CategoryKind.INCOME : CategoryKind.EXPENSE),
    );
  const pickerId = (index: number) => `${id}-cat-${index}`;

  function toggle(index: number, on: boolean) {
    const next = new Set(sel);
    if (on) next.add(index);
    else next.delete(index);
    setSelected(next);
  }

  function rejectRow(index: number, message: string) {
    setMissing(index);
    // Make sure the row is listed: the current view, or else "Needs a category".
    const row = rows.find((r) => r.index === index);
    let list = listed;
    if (row && !list.includes(row)) {
      setView("uncategorised");
      list = viewRows(rows, "uncategorised", sel, picks, showSettled);
    }
    const position = list.findIndex((r) => r.index === index);
    setShown(Math.max(PAGE, position + 1));
    toast.error(message);
    // Let the row render before faulting it.
    requestAnimationFrame(() => fault(pickerId(index)));
  }

  async function submit() {
    if (counts.selected === 0) {
      toast.error("Tick at least one row to import");
      return;
    }
    const gap = firstMissingCategory(rows, sel, picks);
    if (gap) return rejectRow(gap.index, `Choose a category for row ${gap.index + 1}`);
    const chosen = rows.filter((r) => sel.has(r.index) && canChoose(r));
    try {
      const res = await commit.mutateAsync({
        batchId,
        mapping,
        rows: chosen.map((r) => ({ index: r.index, categoryId: picks.get(r.index) ?? "" })),
        saveMapping: remember,
      });
      onDone(res.importedCount);
    } catch (err) {
      // "rows[3].category_id: …" points at the 4th chosen row.
      if (!(err instanceof ConnectError) || err.code !== Code.InvalidArgument) return;
      const m = /^rows\[(\d+)\]/.exec(err.rawMessage);
      const row = m ? chosen[Number(m[1])] : undefined;
      if (row) {
        setMissing(row.index);
        fault(pickerId(row.index));
      }
    }
  }

  if (q.isPending || selected === null) {
    return (
      <Panel>
        <div className="flex flex-col gap-3">
          {[0, 1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-12" />
          ))}
        </div>
      </Panel>
    );
  }
  if (q.isError) {
    return (
      <Panel recess className="max-w-xl">
        <div className="flex flex-col items-start gap-4">
          <p className="text-body-md text-ink">{q.error.rawMessage}</p>
          <Button variant="secondary" onClick={onBack}>
            Change columns
          </Button>
        </div>
      </Panel>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      {savedMapping && (
        <p className="text-body-sm text-ink-soft">
          Read with the columns saved for this wallet.{" "}
          <button
            type="button"
            onClick={onBack}
            className="text-signal-ink underline-offset-4 hover:underline"
          >
            Change columns
          </button>
        </p>
      )}
      <p className="text-body-sm text-ink">
        {counts.total} rows · {counts.selected} ticked
        {counts.duplicates > 0 &&
          ` · ${counts.duplicates} possible ${counts.duplicates === 1 ? "duplicate" : "duplicates"}`}
        {counts.refunded > 0 &&
          ` · ${counts.refunded} refunded ${counts.refunded === 1 ? "purchase" : "purchases"} left out`}
        {counts.merged > 0 &&
          ` · ${counts.merged} ${counts.merged === 1 ? "discount" : "discounts"} merged`}
        {counts.failed > 0 && ` · ${counts.failed} can't be read`}
      </p>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <ToggleGroup
          type="single"
          variant="outline"
          size="sm"
          value={view}
          onValueChange={(v) => v && show(v as ReviewView)}
          aria-label="Show rows"
          className="flex-wrap"
        >
          {(Object.keys(viewLabels) as ReviewView[]).map((v) => (
            <ToggleGroupItem key={v} value={v} disabled={v !== "all" && chips[v] === 0}>
              {viewLabels[v]} <span className="tabular text-ink-soft">{chips[v]}</span>
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
        <div className="flex flex-wrap items-center gap-3">
          {view === "all" && (
            <div className="flex items-center gap-2">
              <Checkbox
                id={`${id}-settled`}
                checked={showSettled}
                onCheckedChange={(v) => {
                  setShowSettled(v === true);
                  setShown(PAGE);
                }}
              />
              <Label htmlFor={`${id}-settled`} className="text-body-sm text-ink">
                Show settled rows
              </Label>
            </div>
          )}
          <Button variant="secondary" size="sm" disabled={allTicked} onClick={() => setTicks(true)}>
            Tick all
          </Button>
          <Button
            variant="secondary"
            size="sm"
            disabled={noneTicked}
            onClick={() => setTicks(false)}
          >
            Untick all
          </Button>
        </div>
      </div>

      <Panel bodyClassName="p-0 sm:p-0" notch="md">
        {listed.length === 0 && folded.length === 0 && (
          <p className="px-4 py-6 text-body-sm text-ink-soft sm:px-6">No rows here.</p>
        )}
        <ul className="flex flex-col">
          {listed.slice(0, shown).map((r) => (
            <li
              key={r.index}
              className="flex flex-col gap-2 border-b border-seam/60 px-4 py-3 last:border-b-0 sm:px-6"
            >
              <div className="flex items-start gap-3">
                <Checkbox
                  aria-label={`Import row ${r.index + 1}`}
                  checked={sel.has(r.index)}
                  disabled={!canChoose(r)}
                  onCheckedChange={(v) => toggle(r.index, v === true)}
                  className="mt-0.5 shrink-0"
                />
                <span className="flex min-w-0 flex-1 flex-col gap-1">
                  <span className="truncate text-body-sm text-ink">
                    {r.error ? r.cells.join(" · ") : r.note || "—"}
                  </span>
                  <span className="flex flex-wrap items-center gap-2">
                    <span className="hud-sm text-ink-soft">
                      Row {r.index + 1}
                      {!r.error && ` · ${formatDay(r.occurredOn)}`}
                    </span>
                    {leftOutBadge(r) && <Badge variant="tint">{leftOutBadge(r)}</Badge>}
                    {r.discountsTransactionId && <Badge variant="outline">Discount</Badge>}
                    {r.categorySource === "rule" && <Badge variant="outline">Rule</Badge>}
                  </span>
                  {rowRelation(r, money) && (
                    <span className="text-body-sm text-ink-soft">{rowRelation(r, money)}</span>
                  )}
                  {r.error && <span className="text-body-sm text-danger">{r.error}</span>}
                </span>
                {!r.error && (
                  <Amount
                    minor={r.kind === TransactionKind.EXPENSE ? -r.amountMinor : r.amountMinor}
                    currency={wallet.currency}
                    minorUnits={units}
                    size="sm"
                    tone={r.mergedInto !== undefined ? "muted" : "auto"}
                    className="shrink-0"
                  />
                )}
              </div>
              {needsCategory(r) && sel.has(r.index) && (
                <div className="flex items-center gap-2 pl-8">
                  {/* Set categories show as a small chip; missing ones get the full picker. */}
                  <div
                    className={
                      picks.get(r.index) && missing !== r.index
                        ? "flex min-w-0 flex-col gap-1"
                        : "flex min-w-0 flex-1 flex-col gap-1"
                    }
                  >
                    <CategoryPicker
                      id={pickerId(r.index)}
                      compact={Boolean(picks.get(r.index)) && missing !== r.index}
                      categories={kindCategories(r.kind)}
                      walletId={wallet.id}
                      value={picks.get(r.index) ?? ""}
                      onChange={(v) => {
                        setPicks(new Map(picks).set(r.index, v));
                        if (missing === r.index) setMissing(null);
                      }}
                    />
                    {missing === r.index && <FieldError>Choose a category</FieldError>}
                  </div>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    aria-label={`Make a rule from row ${r.index + 1}`}
                    title="Make a rule"
                    onClick={() => setRuleFor(r)}
                  >
                    <WandSparkles strokeWidth={1.25} />
                  </Button>
                </div>
              )}
            </li>
          ))}
        </ul>
        {listed.length > shown && (
          <div className="flex justify-center p-4">
            <Button variant="secondary" size="sm" onClick={() => setShown(shown + PAGE)}>
              Show more rows
            </Button>
          </div>
        )}
        {/* Settled rows, one line per kind; each opens the chip that lists them. */}
        {folded.length > 0 && (
          <ul className="flex flex-col border-t border-seam/60 bg-shell-recess/40">
            {folded.map((g) => (
              <li key={g.kind} className="border-b border-seam/60 last:border-b-0">
                <button
                  type="button"
                  onClick={() => show(g.view)}
                  className="flex w-full items-center justify-between gap-3 px-4 py-2.5 text-left hover:bg-shell-recess sm:px-6"
                >
                  <span className="text-body-sm text-ink-soft">{g.label}</span>
                  <span className="hud-sm text-signal-ink">Show</span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </Panel>

      {/* Pinned while the list scrolls, so importing never needs a trip to the end. */}
      <div className="glass sticky bottom-[calc(env(safe-area-inset-bottom)+1rem)] z-30 flex items-center justify-between gap-3 rounded-full py-2 pr-2 pl-5">
        <span className="min-w-0 truncate text-label text-ink">
          {counts.selected} ticked
          {chips.uncategorised > 0 && (
            <span className="hidden text-ink-soft sm:inline">
              {" "}
              · {chips.uncategorised} need a category
            </span>
          )}
        </span>
        <Button onClick={submit} disabled={commit.isPending} className="shrink-0">
          Import {counts.selected} {counts.selected === 1 ? "transaction" : "transactions"}
        </Button>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-2">
          <Checkbox
            id={`${id}-remember`}
            checked={remember}
            onCheckedChange={(v) => setRemember(v === true)}
          />
          <Label htmlFor={`${id}-remember`} className="text-body-sm text-ink">
            Remember these columns for {wallet.name}
          </Label>
        </div>
        <Button variant="ghost" onClick={onBack}>
          Change columns
        </Button>
      </div>
      <p className="text-body-sm text-ink-soft">
        Rules fill in categories from the note on every import.{" "}
        <Link
          href="/settings/import-rules"
          className="text-signal-ink underline-offset-4 hover:underline"
        >
          Manage rules
        </Link>
      </p>

      {ruleFor && (
        <RuleDialog
          row={ruleFor}
          walletId={wallet.id}
          categories={kindCategories(ruleFor.kind)}
          initialCategory={picks.get(ruleFor.index) ?? ""}
          onClose={() => setRuleFor(null)}
          onCreated={(pattern, categoryId) =>
            setPicks(
              applyRule(
                rows,
                new Map(picks).set(ruleFor.index, categoryId),
                pattern,
                categoryId,
                ruleFor.kind,
              ),
            )
          }
        />
      )}
    </div>
  );
}

/** "When the note contains … use …", prefilled from a row. */
function RuleDialog({
  row,
  walletId,
  categories,
  initialCategory,
  onClose,
  onCreated,
}: {
  row: ImportRow;
  walletId: string;
  categories: ReturnType<typeof useCategories>["categories"];
  initialCategory: string;
  onClose: () => void;
  onCreated: (pattern: string, categoryId: string) => void;
}) {
  const id = useId();
  const [pattern, setPattern] = useState(row.note);
  const [categoryId, setCategoryId] = useState(initialCategory);
  const [errors, setErrors] = useState<{ pattern?: string; cat?: string }>({});
  const create = useMutation(ImportService.method.createCategoryRule);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    if (!pattern.trim()) {
      setErrors({ pattern: "Enter the text to look for" });
      return fault(`${id}-pattern`);
    }
    if (!categoryId) {
      setErrors({ cat: "Choose a category" });
      return fault(`${id}-cat`);
    }
    try {
      await create.mutateAsync({ pattern, categoryId });
    } catch {
      return; // toasted
    }
    toast.success("Rule saved");
    onCreated(pattern, categoryId);
    onClose();
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Make a rule</DialogTitle>
          <DialogDescription>
            Rows whose note contains this text get this category, now and on later imports.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={save} className="flex flex-col gap-4">
          <Field label="When the note contains" htmlFor={`${id}-pattern`} error={errors.pattern}>
            <Input
              id={`${id}-pattern`}
              value={pattern}
              maxLength={100}
              onChange={(e) => {
                setPattern(e.target.value);
                setErrors({});
              }}
            />
          </Field>
          <Field label="Use the category" htmlFor={`${id}-cat`} error={errors.cat}>
            <CategoryPicker
              id={`${id}-cat`}
              categories={categories}
              walletId={walletId}
              value={categoryId}
              onChange={(v) => {
                setCategoryId(v);
                setErrors({});
              }}
            />
          </Field>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={create.isPending}>
              Save rule
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
