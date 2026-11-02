/** What a backup file holds, read in the browser before it's restored. */
export type BackupSummary = {
  household: string;
  exportedAt?: Date;
  counts: {
    wallets: number;
    categories: number;
    labels: number;
    transactions: number;
    transfers: number;
    budgets: number;
    recurringRules: number;
    importRules: number;
  };
};

const isGzip = (b: Uint8Array) => b.length >= 2 && b[0] === 0x1f && b[1] === 0x8b;

async function gunzip(b: Uint8Array): Promise<string> {
  const stream = new Blob([b as BlobPart]).stream().pipeThrough(new DecompressionStream("gzip"));
  return new Response(stream).text();
}

const count = (v: unknown) => (Array.isArray(v) ? v.length : 0);

/** Reads a backup's header and row counts. Throws with a message for the user if it isn't one. */
export async function readBackupSummary(content: Uint8Array): Promise<BackupSummary> {
  let doc: Record<string, unknown>;
  try {
    const text = isGzip(content) ? await gunzip(content) : new TextDecoder().decode(content);
    doc = JSON.parse(text);
  } catch {
    throw new Error("This isn't a Ledger/01 backup.");
  }
  if (doc?.format !== "ledger-01-backup") throw new Error("This isn't a Ledger/01 backup.");
  const exported = typeof doc.exported_at === "string" ? new Date(doc.exported_at) : undefined;
  const household = doc.household as { name?: unknown } | undefined;
  return {
    household: typeof household?.name === "string" ? household.name : "",
    exportedAt: exported && !Number.isNaN(exported.getTime()) ? exported : undefined,
    counts: {
      wallets: count(doc.wallets),
      categories: count(doc.categories),
      labels: count(doc.labels),
      transactions: count(doc.transactions),
      transfers: count(doc.transfers),
      budgets: count(doc.budgets),
      recurringRules: count(doc.recurring_rules),
      importRules: count(doc.import_rules),
    },
  };
}
