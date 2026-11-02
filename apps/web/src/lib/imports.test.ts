import { describe, expect, test } from "bun:test";
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  ImportDuplicate,
  ImportMappingSchema,
  type ImportRow,
  ImportRowSchema,
  TransactionKind,
} from "@ledger-01/api";
import {
  amountMode,
  applyRule,
  checkMapping,
  columnFromValue,
  columnOptions,
  columnValue,
  firstMissingCategory,
  formatExample,
  initialCategories,
  initialSelection,
  inView,
  leftOutBadge,
  needsCategory,
  reviewCounts,
  rowRelation,
  settledGroups,
  settledKind,
  tableStartOptions,
  viewCounts,
  viewRows,
} from "./imports";

const row = (init: MessageInitShape<typeof ImportRowSchema>): ImportRow =>
  create(ImportRowSchema, { kind: TransactionKind.EXPENSE, ...init });

const rows: ImportRow[] = [
  row({ index: 0, note: "Opening deposit", kind: TransactionKind.INCOME }),
  row({ index: 1, note: "Corner Grocer", duplicate: true, categoryId: "groceries" }),
  row({ index: 2, note: "City Transit Card", categoryId: "transport" }),
  row({ index: 3, error: '"yesterday" isn\'t a date' }),
  row({ index: 4, note: "Corner Grocer #2" }),
];

describe("columns", () => {
  test("labels by header and example value", () => {
    const sample = [
      ["Date", "Description", "Amount"],
      ["2026-09-01", "Corner Grocer", "-45.20"],
    ];
    expect(columnOptions(sample, true)).toEqual([
      { value: "0", label: "Date · 2026-09-01" },
      { value: "1", label: "Description · Corner Grocer" },
      { value: "2", label: "Amount · -45.20" },
    ]);
    expect(columnOptions(sample, false)[1]).toEqual({
      value: "1",
      label: "Column 2 · Description",
    });
  });

  test("select values round-trip, with none", () => {
    expect(columnValue(undefined)).toBe("none");
    expect(columnFromValue(columnValue(3))).toBe(3);
    expect(columnFromValue("none")).toBeUndefined();
  });

  test("amount mode and checks", () => {
    const single = create(ImportMappingSchema, { amountColumn: 2 });
    const split = create(ImportMappingSchema, { debitColumn: 2, creditColumn: 3 });
    expect(amountMode(single)).toBe("single");
    expect(amountMode(split)).toBe("split");
    expect(amountMode(create(ImportMappingSchema, {}))).toBe("single");
    expect(checkMapping(single, "single")).toBeNull();
    expect(checkMapping(split, "single")?.field).toBe("amount");
    expect(checkMapping(single, "split")?.field).toBe("debit");
    expect(checkMapping(create(ImportMappingSchema, { creditColumn: 1 }), "split")).toBeNull();
  });

  test("date format examples", () => {
    expect(formatExample("dd/mm/yyyy")).toBe("31/12/2026");
    expect(formatExample("yyyy-mm-dd")).toBe("2026-12-31");
    expect(formatExample("mm/dd/yy")).toBe("12/31/26");
  });
});

describe("review", () => {
  test("ticks importable rows that aren't duplicates", () => {
    expect([...initialSelection(rows)].sort()).toEqual([0, 2, 4]);
  });

  test("starts from the suggested categories", () => {
    expect(initialCategories(rows)).toEqual(
      new Map([
        [1, "groceries"],
        [2, "transport"],
      ]),
    );
  });

  test("a new rule fills rows of its kind without a category", () => {
    const picks = applyRule(
      rows,
      initialCategories(rows),
      " grocer ",
      "food",
      TransactionKind.EXPENSE,
    );
    expect(picks.get(1)).toBe("groceries"); // kept
    expect(picks.get(4)).toBe("food");
    expect(picks.get(0)).toBeUndefined();
    expect(applyRule(rows, new Map(), "deposit", "gifts", TransactionKind.EXPENSE).size).toBe(0);
    expect(applyRule(rows, new Map(), "  ", "gifts", TransactionKind.EXPENSE).size).toBe(0);
  });

  test("counts and the first missing category", () => {
    const selected = new Set([0, 2, 3, 4]);
    expect(reviewCounts(rows, selected)).toEqual({
      total: 5,
      selected: 3,
      duplicates: 1,
      merged: 0,
      refunded: 0,
      failed: 1,
    });
    expect(firstMissingCategory(rows, selected, initialCategories(rows))?.index).toBe(0);
    expect(firstMissingCategory(rows, new Set([2]), initialCategories(rows))).toBeUndefined();
  });
});

describe("table start", () => {
  const sample = [
    ["Banco del Sur", "", ""],
    ["Titular", "PEREZ GOMEZ ANA", ""],
    ["Fecha", "Concepto", "Importe"],
    ["01/10/2026", "Supermercado Central", "-45,20"],
  ];

  test("rows are offered by their first filled cells", () => {
    expect(tableStartOptions(sample).map((o) => o.label)).toEqual([
      "Row 1 · Banco del Sur",
      "Row 2 · Titular · PEREZ GOMEZ ANA",
      "Row 3 · Fecha · Concepto · Importe",
      "Row 4 · 01/10/2026 · Supermercado Central · -45,20",
    ]);
  });

  test("columns are named from the table's header row", () => {
    expect(columnOptions(sample.slice(2), true)[1]).toEqual({
      value: "1",
      label: "Concepto · Supermercado Central",
    });
  });
});

describe("matches and discounts", () => {
  const money = (m: bigint) => (Number(m) / 100).toFixed(2);
  const purchase = row({
    index: 0,
    note: "COMPRA MERCADO SOL",
    amountMinor: 48361n,
    discountMinor: 1639n,
    discountNotes: ["REDIVA 19210MERCADO SOL"],
    categoryId: "food",
  });
  const merged = row({ index: 1, note: "REDIVA 19210MERCADO SOL", mergedInto: 0 });
  const earlier = row({
    index: 2,
    note: "REDIVA 19210KIOSCO AZUL",
    amountMinor: 1639n,
    discountsTransactionId: "tx-1",
    discountsNote: "Snacks",
    discountsDate: "2026-10-31",
  });
  const renamed = row({
    index: 3,
    duplicate: true,
    duplicateReason: ImportDuplicate.IMPORTED,
    matchNote: "Snacks",
    matchDate: "2026-10-01",
  });
  const all = [purchase, merged, earlier, renamed];

  test("merged discounts can't be ticked; discounts of earlier purchases need no category", () => {
    expect([...initialSelection(all)]).toEqual([0, 2]);
    expect(needsCategory(earlier)).toBe(false);
    expect(firstMissingCategory(all, new Set([0, 2]), initialCategories(all))).toBeUndefined();
    expect(reviewCounts(all, new Set([0, 1, 2])).merged).toBe(1);
    expect(reviewCounts(all, new Set([0, 1, 2])).selected).toBe(2);
  });

  test("each row says what it matched or merged", () => {
    expect(rowRelation(purchase, money)).toBe(
      "Includes a discount of 16.39: REDIVA 19210MERCADO SOL",
    );
    expect(rowRelation(merged, money)).toBe("Discount counted in row 1");
    expect(rowRelation(earlier, money)).toBe(
      "Discount: lowers “Snacks” on Sat, Oct 31, 2026 by 16.39",
    );
    expect(rowRelation(renamed, money)).toBe("Imported before; now “Snacks” on Thu, Oct 1, 2026");
  });
});

describe("refunds", () => {
  const money = (m: bigint) => (Number(m) / 100).toFixed(2);
  const charge = row({
    index: 0,
    note: "COMPRA PAY*TRANS*BUS",
    amountMinor: 740n,
    categoryId: "taxi",
  });
  const second = row({ index: 1, note: "COMPRA PAY*TRANS*BUS", amountMinor: 740n, refundedBy: 2 });
  const refund = row({ index: 2, note: "DEVOLUCION PAY*TRANS*BUS", refundOf: 1 });
  const later = row({ index: 3, refundsNote: "Shoes", refundsDate: "2026-10-30" });

  test("a refunded purchase and its refund start unticked", () => {
    expect([...initialSelection([charge, second, refund, later])]).toEqual([0]);
    expect(reviewCounts([charge, second, refund, later], new Set([0])).refunded).toBe(1);
  });

  test("badges and relations", () => {
    expect(leftOutBadge(second)).toBe("Refunded");
    expect(leftOutBadge(refund)).toBe("Refund");
    expect(rowRelation(second, money)).toBe("Refunded in full by row 3");
    expect(rowRelation(refund, money)).toBe("Refund of row 2");
    expect(rowRelation(later, money)).toBe(
      "Refunds “Shoes” on Fri, Oct 30, 2026, already in the wallet",
    );
  });
});

describe("review views", () => {
  const open = row({ index: 0, note: "COMPRA KIOSCO AZUL", categoryId: "food" });
  const uncategorised = row({ index: 1, note: "COMPRA FERRETERIA" });
  const imported = row({ index: 2, duplicate: true, duplicateReason: ImportDuplicate.IMPORTED });
  const similar = row({ index: 3, duplicate: true, duplicateReason: ImportDuplicate.SIMILAR });
  const refunded = row({ index: 4, refundedBy: 5 });
  const refund = row({ index: 5, refundOf: 4 });
  const merged = row({ index: 6, mergedInto: 0 });
  const broken = row({ index: 7, error: "bad date" });
  const all = [open, uncategorised, imported, similar, refunded, refund, merged, broken];
  const selected = initialSelection(all);
  const picks = initialCategories(all);
  const indexes = (rs: ImportRow[]) => rs.map((r) => r.index);

  test("settled kinds", () => {
    expect(all.map(settledKind)).toEqual([
      undefined,
      undefined,
      "imported",
      "duplicate",
      "refund",
      "refund",
      undefined,
      "unreadable",
    ]);
  });

  test("each view lists its rows; merged discounts never list", () => {
    expect(indexes(viewRows(all, "all", selected, picks, false))).toEqual([0, 1]);
    expect(indexes(viewRows(all, "all", selected, picks, true))).toEqual([0, 1, 2, 3, 4, 5, 7]);
    expect(indexes(viewRows(all, "ticked", selected, picks, false))).toEqual([0, 1]);
    expect(indexes(viewRows(all, "uncategorised", selected, picks, false))).toEqual([1]);
    expect(indexes(viewRows(all, "duplicates", selected, picks, false))).toEqual([2, 3]);
    expect(indexes(viewRows(all, "left-out", selected, picks, false))).toEqual([4, 5, 7]);
    expect(inView(merged, "all", selected, picks)).toBe(false);
  });

  test("chip counts and the folded lines", () => {
    expect(viewCounts(all, selected, picks)).toEqual({
      all: 7,
      ticked: 2,
      uncategorised: 1,
      duplicates: 2,
      "left-out": 3,
    });
    expect(settledGroups(all).map((g) => [g.label, g.view])).toEqual([
      ["1 already imported", "duplicates"],
      ["1 possible duplicate", "duplicates"],
      ["2 refund rows left out", "left-out"],
      ["1 can't be read", "left-out"],
    ]);
  });
});
