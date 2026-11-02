import { Bucket } from "@ledger-01/api";
import { addMonths, differenceInCalendarDays, endOfMonth, format, startOfMonth } from "date-fns";
import { type Period, toISODate } from "./period";

export type BucketName = "day" | "week" | "month";

export const bucketProto: Record<BucketName, Bucket> = {
  day: Bucket.DAY,
  week: Bucket.WEEK,
  month: Bucket.MONTH,
};

const toDate = (s: string) => {
  const [y = 1970, m = 1, d = 1] = s.split("-").map(Number);
  return new Date(y, m - 1, d);
};

/** Days for up to ~2 months, weeks up to ~6 months, months beyond. */
export function defaultBucket(p: Period): BucketName {
  const days = differenceInCalendarDays(toDate(p.to), toDate(p.from)) + 1;
  if (days <= 62) return "day";
  if (days <= 186) return "week";
  return "month";
}

/** Calendar months each grouping spans, so every choice draws a useful number of points. */
export const bucketSpanMonths: Record<BucketName, number> = { day: 1, week: 3, month: 12 };

/** The whole-month period for a grouping, ending with the month that contains `anchor`. */
export function periodForBucket(bucket: BucketName, anchor: string): Period {
  const end = endOfMonth(toDate(anchor));
  const start = startOfMonth(addMonths(end, 1 - bucketSpanMonths[bucket]));
  return { from: toISODate(start), to: toISODate(end) };
}

export function parseBucket(value: string | null, p: Period): BucketName {
  return value === "day" || value === "week" || value === "month" ? value : defaultBucket(p);
}

/** Short axis label and full tooltip label for a bucket's first day. */
export function bucketLabels(
  date: string,
  bucket: BucketName,
): { label: string; tooltipLabel: string } {
  const d = toDate(date);
  switch (bucket) {
    case "month":
      return { label: format(d, "MMM"), tooltipLabel: format(d, "MMMM yyyy") };
    case "week":
      return { label: format(d, "MMM d"), tooltipLabel: `Week of ${format(d, "MMM d, yyyy")}` };
    default:
      return { label: format(d, "MMM d"), tooltipLabel: format(d, "EEE, MMM d, yyyy") };
  }
}
