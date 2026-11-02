import { describe, expect, test } from "bun:test";
import { gzipSync } from "node:zlib";
import { readBackupSummary } from "./backup";

const doc = {
  format: "ledger-01-backup",
  version: 1,
  exported_at: "2026-10-31T12:00:00Z",
  household: { name: "Test household" },
  wallets: [{}, {}],
  transactions: [{}, {}, {}],
  import_rules: [{}],
};

describe("readBackupSummary", () => {
  test("reads a gzipped backup", async () => {
    const s = await readBackupSummary(new Uint8Array(gzipSync(JSON.stringify(doc))));
    expect(s.household).toBe("Test household");
    expect(s.exportedAt?.toISOString()).toBe("2026-10-31T12:00:00.000Z");
    expect(s.counts).toMatchObject({ wallets: 2, transactions: 3, importRules: 1, budgets: 0 });
  });

  test("reads plain JSON", async () => {
    const s = await readBackupSummary(new TextEncoder().encode(JSON.stringify(doc)));
    expect(s.counts.wallets).toBe(2);
  });

  test("rejects other files", async () => {
    await expect(readBackupSummary(new TextEncoder().encode("Date,Amount"))).rejects.toThrow(
      "isn't a Ledger/01 backup",
    );
    await expect(
      readBackupSummary(new TextEncoder().encode(JSON.stringify({ format: "other" }))),
    ).rejects.toThrow("isn't a Ledger/01 backup");
  });
});
