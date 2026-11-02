import { parseAmount } from "@ledger-01/ui/lib/money";
import { type BucketName, parseBucket } from "@/lib/buckets";
import { type Period, parsePeriod } from "@/lib/period";

/** Report filters, as they live in the URL. */
export type ReportState = {
  period: Period;
  bucket: BucketName;
  walletIds: string[];
  categoryIds: string[];
  /** Report currency: the URL's, else the page's default. */
  currency: string;
  /** Note search and amount bounds as typed (amounts in the report currency). */
  note: string;
  min: string;
  max: string;
};

const list = (v: string | null) => (v ? v.split(",").filter(Boolean) : []);

/** Reads report filters from the URL; the currency falls back to `defaultCurrency`. */
export function parseReportState(params: URLSearchParams, defaultCurrency: string): ReportState {
  const period = parsePeriod(params);
  return {
    period,
    bucket: parseBucket(params.get("bucket"), period),
    walletIds: list(params.get("wallets")),
    categoryIds: list(params.get("categories")),
    currency: params.get("currency") || defaultCurrency,
    note: params.get("note") ?? "",
    min: params.get("min") ?? "",
    max: params.get("max") ?? "",
  };
}

/**
 * The URL query for report filters. The bucket and the currency are only written when the URL
 * already had them (or the bucket was just picked), so defaults stay out of shared links.
 */
export function reportQuery(
  s: ReportState,
  current: URLSearchParams,
  bucketPicked: boolean,
): URLSearchParams {
  const q = new URLSearchParams({ from: s.period.from, to: s.period.to });
  if (current.get("bucket") || bucketPicked) q.set("bucket", s.bucket);
  if (s.walletIds.length) q.set("wallets", s.walletIds.join(","));
  if (s.categoryIds.length) q.set("categories", s.categoryIds.join(","));
  if (current.get("currency")) q.set("currency", s.currency);
  if (s.note.trim()) q.set("note", s.note.trim());
  if (s.min) q.set("min", s.min);
  if (s.max) q.set("max", s.max);
  return q;
}

/** An amount bound in minor units; empty or half-typed input means no bound. */
export function amountBound(v: string, minorUnits: number): bigint | undefined {
  return v ? (parseAmount(v, minorUnits) ?? undefined) : undefined;
}
