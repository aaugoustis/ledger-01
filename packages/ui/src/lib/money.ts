// Money helpers. Amounts are integer minor units (bigint, matching protobuf int64) plus the
// currency's number of decimal places. Never use floats for money.

export type SignDisplay = "auto" | "always" | "never";

const pow10 = (n: number) => 10n ** BigInt(n);

/** Exact decimal string for a minor-unit amount: (123456n, 2) -> "1234.56". */
export function minorToDecimal(amount: bigint, minorUnits: number): string {
  const neg = amount < 0n;
  const abs = neg ? -amount : amount;
  if (minorUnits === 0) return `${neg ? "-" : ""}${abs}`;
  const scale = pow10(minorUnits);
  const whole = abs / scale;
  const frac = (abs % scale).toString().padStart(minorUnits, "0");
  return `${neg ? "-" : ""}${whole}.${frac}`;
}

/**
 * Parses user input like "1,234.5" or "1234.56" into minor units. Returns null for anything
 * that isn't a plain non-negative decimal with at most `minorUnits` decimals.
 */
export function parseAmount(input: string, minorUnits: number): bigint | null {
  const s = input.trim().replace(/,/g, "");
  if (!/^\d+(\.\d*)?$|^\.\d+$/.test(s)) return null;
  const [whole = "0", frac = ""] = s.split(".");
  if (frac.length > minorUnits) return null;
  return BigInt(whole || "0") * pow10(minorUnits) + BigInt(frac.padEnd(minorUnits, "0") || "0");
}

const formatters = new Map<number, Intl.NumberFormat>();
function formatter(minorUnits: number) {
  let f = formatters.get(minorUnits);
  if (!f) {
    f = new Intl.NumberFormat("en-US", {
      minimumFractionDigits: minorUnits,
      maximumFractionDigits: minorUnits,
      signDisplay: "never",
    });
    formatters.set(minorUnits, f);
  }
  return f;
}

/**
 * Formats an amount the way the app shows money: "+1,234.56 USD", "−42.05 USD", "0.00 USD".
 * Uses a true minus sign (U+2212) so negatives align with plus signs in mono type.
 */
export function formatMoney(
  amount: bigint,
  currency: string,
  minorUnits: number,
  opts: { sign?: SignDisplay; code?: boolean } = {},
): string {
  const { sign = "auto", code = true } = opts;
  // Intl formats decimal strings exactly (no float rounding).
  const digits = formatter(minorUnits).format(
    minorToDecimal(amount < 0n ? -amount : amount, minorUnits) as unknown as number,
  );
  let prefix = "";
  if (amount < 0n && sign !== "never") prefix = "−";
  else if (amount > 0n && sign === "always") prefix = "+";
  return `${prefix}${digits}${code ? ` ${currency}` : ""}`;
}

/** Plain input-friendly value without grouping: 123456n, 2 -> "1234.56". */
export function formatAmountInput(amount: bigint | null, minorUnits: number): string {
  return amount === null ? "" : minorToDecimal(amount, minorUnits);
}
