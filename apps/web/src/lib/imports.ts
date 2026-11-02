import {
  ImportDuplicate,
  type ImportMapping,
  type ImportRow,
  type TransactionKind,
} from "@ledger-01/api";
import { formatDay } from "./period";

/** Select value for "no column" (Radix selects can't use an empty value). */
export const NONE = "none";

export const columnValue = (c: number | undefined) => (c === undefined ? NONE : String(c));
export const columnFromValue = (v: string) => (v === NONE ? undefined : Number(v));

/**
 * Choices for where the table starts, each row shown by its first filled cells, e.g. "Row 6 ·
 * Date · Description · Amount", so the bank's logo and account lines above it are easy to skip.
 */
export function tableStartOptions(sample: string[][]) {
  return sample.map((cells, i) => {
    const text = cells
      .map((c) => c.trim())
      .filter(Boolean)
      .slice(0, 4)
      .join(" · ");
    const label = text.length > 60 ? `${text.slice(0, 59)}…` : text;
    return { value: String(i), label: `Row ${i + 1}${label ? ` · ${label}` : ""}` };
  });
}

/**
 * Column choices labelled by header (or position) and an example value, e.g. "Description ·
 * Corner Grocer" or "Column 3 · -45.20". Pass the sample from the table's first row on.
 */
export function columnOptions(sample: string[][], hasHeader: boolean) {
  const width = Math.max(0, ...sample.map((r) => r.length));
  const first = hasHeader ? sample[1] : sample[0];
  return Array.from({ length: width }, (_, c) => {
    const name = (hasHeader && sample[0]?.[c]) || `Column ${c + 1}`;
    const example = first?.[c];
    return { value: String(c), label: example ? `${name} · ${example}` : name };
  });
}

export type AmountMode = "single" | "split";

/** "split" when the mapping reads debit/credit columns instead of one amount column. */
export function amountMode(
  m: Pick<ImportMapping, "amountColumn" | "debitColumn" | "creditColumn">,
) {
  return m.amountColumn === undefined &&
    (m.debitColumn !== undefined || m.creditColumn !== undefined)
    ? "split"
    : "single";
}

export type MappingFault = { field: "amount" | "debit"; message: string };

/** Checks what the server would reject first, naming the control at fault. */
export function checkMapping(m: ImportMapping, mode: AmountMode): MappingFault | null {
  if (mode === "single" && m.amountColumn === undefined)
    return { field: "amount", message: "Choose the amount column" };
  if (mode === "split" && m.debitColumn === undefined && m.creditColumn === undefined)
    return { field: "debit", message: "Choose the debit or the credit column" };
  return null;
}

/** A row can be ticked unless it can't be read or is a discount counted in another row. */
export const canChoose = (r: ImportRow) => !r.error && r.mergedInto === undefined;

/** Ticked rows need a category, except discounts that lower an earlier purchase. */
export const needsCategory = (r: ImportRow) => canChoose(r) && !r.discountsTransactionId;

/** A purchase refunded in full and its refund, in the file or against the wallet. */
export const isRefundPair = (r: ImportRow) =>
  r.refundedBy !== undefined || r.refundOf !== undefined || Boolean(r.refundsNote);

/**
 * Rows ticked when the review opens: the ones that can be imported, aren't duplicates and aren't
 * a purchase refunded in full (or its refund), which are left out as if neither happened.
 */
export function initialSelection(rows: ImportRow[]): Set<number> {
  return new Set(
    rows.filter((r) => canChoose(r) && !r.duplicate && !isRefundPair(r)).map((r) => r.index),
  );
}

/**
 * Rows that need no decision: imported before, possible duplicates, refund pairs and rows that
 * can't be read. The review folds them into one line per kind.
 */
export type SettledKind = "imported" | "duplicate" | "refund" | "unreadable";

export function settledKind(r: ImportRow): SettledKind | undefined {
  if (r.error) return "unreadable";
  if (r.duplicateReason === ImportDuplicate.IMPORTED) return "imported";
  if (r.duplicate) return "duplicate";
  if (isRefundPair(r)) return "refund";
  return undefined;
}

/** A discount merged into its purchase isn't listed: the purchase row says so. */
export const isFolded = (r: ImportRow) => r.mergedInto !== undefined;

/** The review's filter chips. */
export type ReviewView = "all" | "ticked" | "uncategorised" | "duplicates" | "left-out";

const settledView: Record<SettledKind, ReviewView> = {
  imported: "duplicates",
  duplicate: "duplicates",
  refund: "left-out",
  unreadable: "left-out",
};

/** Whether a row belongs to a view (settled rows included). */
export function inView(
  r: ImportRow,
  view: ReviewView,
  selected: Set<number>,
  picks: Map<number, string>,
): boolean {
  if (isFolded(r)) return false;
  const kind = settledKind(r);
  switch (view) {
    case "ticked":
      return selected.has(r.index) && canChoose(r);
    case "uncategorised":
      return selected.has(r.index) && needsCategory(r) && !picks.get(r.index);
    case "duplicates":
    case "left-out":
      return kind !== undefined && settledView[kind] === view;
    default:
      return true;
  }
}

/** The rows a view lists. "All" leaves settled rows out unless showSettled. */
export function viewRows(
  rows: ImportRow[],
  view: ReviewView,
  selected: Set<number>,
  picks: Map<number, string>,
  showSettled: boolean,
): ImportRow[] {
  return rows.filter(
    (r) => inView(r, view, selected, picks) && (view !== "all" || showSettled || !settledKind(r)),
  );
}

/** How many rows each chip holds ("all" counts every listed row, settled ones included). */
export function viewCounts(
  rows: ImportRow[],
  selected: Set<number>,
  picks: Map<number, string>,
): Record<ReviewView, number> {
  const views: ReviewView[] = ["all", "ticked", "uncategorised", "duplicates", "left-out"];
  return Object.fromEntries(
    views.map((v) => [v, rows.filter((r) => inView(r, v, selected, picks)).length]),
  ) as Record<ReviewView, number>;
}

const settledLabels: Record<SettledKind, (n: number) => string> = {
  imported: (n) => `${n} already imported`,
  duplicate: (n) => `${n} possible ${n === 1 ? "duplicate" : "duplicates"}`,
  refund: (n) => `${n} refund ${n === 1 ? "row" : "rows"} left out`,
  unreadable: (n) => `${n} can't be read`,
};

/** One line per settled kind for the "All" view, with the chip that lists them. */
export function settledGroups(rows: ImportRow[]) {
  const order: SettledKind[] = ["imported", "duplicate", "refund", "unreadable"];
  return order
    .map((kind) => {
      const count = rows.filter((r) => !isFolded(r) && settledKind(r) === kind).length;
      return { kind, count, label: settledLabels[kind](count), view: settledView[kind] };
    })
    .filter((g) => g.count > 0);
}

/** The badge for a row that starts unticked: a duplicate or a refund pair. */
export function leftOutBadge(r: ImportRow): string | undefined {
  if (r.refundedBy !== undefined) return "Refunded";
  if (r.refundOf !== undefined || r.refundsNote) return "Refund";
  return duplicateBadge(r);
}

/** The badge for a duplicate row. */
export function duplicateBadge(r: ImportRow): string | undefined {
  if (!r.duplicate) return undefined;
  return r.duplicateReason === ImportDuplicate.IMPORTED ? "Already imported" : "Possible duplicate";
}

/**
 * What a row matched or was merged with, in plain words, e.g. "Imported before; now “Snacks” on
 * Thu, Oct 1, 2026" or "Includes a discount of 16.39: REDIVA 19210MERCADO SOL".
 */
export function rowRelation(r: ImportRow, money: (minor: bigint) => string): string | undefined {
  if (r.mergedInto !== undefined) return `Discount counted in row ${r.mergedInto + 1}`;
  if (r.refundedBy !== undefined) return `Refunded in full by row ${r.refundedBy + 1}`;
  if (r.refundOf !== undefined) return `Refund of row ${r.refundOf + 1}`;
  if (r.refundsNote)
    return `Refunds “${r.refundsNote}” on ${formatDay(r.refundsDate)}, already in the wallet`;
  if (r.discountsTransactionId)
    return `Discount: lowers “${r.discountsNote}” on ${formatDay(r.discountsDate)} by ${money(r.amountMinor)}`;
  const on = r.matchDate ? formatDay(r.matchDate) : "";
  switch (r.duplicateReason) {
    case ImportDuplicate.IMPORTED:
      return `Imported before; now “${r.matchNote}” on ${on}`;
    case ImportDuplicate.SAME:
      return `Already in the wallet on ${on}`;
    case ImportDuplicate.SIMILAR:
      return `Same amount as “${r.matchNote}” on ${on}`;
    case ImportDuplicate.FILE:
      return "Repeated earlier in the file";
  }
  if (r.discountMinor > 0n)
    return `Includes a discount of ${money(r.discountMinor)}: ${r.discountNotes.join(", ")}`;
  return undefined;
}

/** Category picks to start from: the suggested ones. */
export function initialCategories(rows: ImportRow[]): Map<number, string> {
  return new Map(rows.filter((r) => r.categoryId).map((r) => [r.index, r.categoryId]));
}

/**
 * Applies a new "note contains" rule the way the server would on the next upload: to rows of the
 * rule's kind that have no category yet.
 */
export function applyRule(
  rows: ImportRow[],
  picks: Map<number, string>,
  pattern: string,
  categoryId: string,
  kind: TransactionKind,
): Map<number, string> {
  const needle = pattern.trim().toLowerCase();
  const next = new Map(picks);
  if (!needle) return next;
  for (const r of rows) {
    if (
      needsCategory(r) &&
      r.kind === kind &&
      !next.get(r.index) &&
      r.note.toLowerCase().includes(needle)
    )
      next.set(r.index, categoryId);
  }
  return next;
}

/** The review summary: rows in the file, ticked, duplicates and rows that can't be imported. */
export function reviewCounts(rows: ImportRow[], selected: Set<number>) {
  return {
    total: rows.length,
    selected: rows.filter((r) => selected.has(r.index) && canChoose(r)).length,
    duplicates: rows.filter((r) => r.duplicate).length,
    merged: rows.filter((r) => r.mergedInto !== undefined).length,
    refunded: rows.filter((r) => r.refundedBy !== undefined).length,
    failed: rows.filter((r) => r.error).length,
  };
}

/** The first ticked row without a category, if any. */
export function firstMissingCategory(
  rows: ImportRow[],
  selected: Set<number>,
  picks: Map<number, string>,
): ImportRow | undefined {
  return rows.find((r) => selected.has(r.index) && needsCategory(r) && !picks.get(r.index));
}

/** An example date in a format, e.g. "dd/mm/yyyy" → "31/12/2026". */
export function formatExample(format: string): string {
  return format.replace("yyyy", "2026").replace("yy", "26").replace("mm", "12").replace("dd", "31");
}

/** Files above this are refused before uploading (the server enforces the same cap). */
export const MAX_IMPORT_BYTES = 4 * 1024 * 1024;
