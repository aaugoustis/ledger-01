// The three-field currency converter used for cross-currency transfers.
//
// Fields: `from` (amount leaving the source wallet), `to` (amount arriving in the target wallet)
// and `rate` (target units per 1 source unit). When the user edits one field, the least recently
// touched of the other two is recomputed from the two most recent ones:
//   to = from × rate,   from = to ÷ rate,   rate = to ÷ from
// Before anything is touched the order (oldest first) is rate, to, from, so typing both amounts
// derives the rate — the usual case when copying numbers from a bank statement.

import { minorToDecimal, parseAmount } from "./money";

export type FxField = "from" | "to" | "rate";

export type FxState = {
  from: string;
  to: string;
  /** What the rate input shows (6 decimals when computed). */
  rate: string;
  /** Full-precision rate (10 decimals) sent to the API; equals `rate` when typed by the user. */
  ratePrecise: string;
  /** Touch order, oldest first. Always holds all three fields. */
  order: FxField[];
};

export type FxCurrencies = { fromMinorUnits: number; toMinorUnits: number };

/** Decimal places kept for rates in storage (numeric(20,10)). */
export const RATE_SCALE = 10;
/** Decimal places shown in the rate input when the rate is computed. */
export const RATE_DISPLAY = 6;

const SCALE = 10n ** BigInt(RATE_SCALE);
const pow10 = (n: number) => 10n ** BigInt(n);

export function initialFxState(): FxState {
  return { from: "", to: "", rate: "", ratePrecise: "", order: ["rate", "to", "from"] };
}

/** Parses a positive decimal rate into a bigint scaled by 10^RATE_SCALE. */
export function parseRate(input: string): bigint | null {
  const s = input.trim().replace(/,/g, "");
  if (!/^\d+(\.\d*)?$|^\.\d+$/.test(s)) return null;
  const [whole = "0", frac = ""] = s.split(".");
  const scaled =
    BigInt(whole || "0") * SCALE + BigInt(frac.slice(0, RATE_SCALE).padEnd(RATE_SCALE, "0") || "0");
  return scaled > 0n ? scaled : null;
}

/** Rounds n / d to the nearest integer, ties to even (banker's rounding). n, d > 0. */
export function divRoundHalfEven(n: bigint, d: bigint): bigint {
  const q = n / d;
  const r = n % d;
  const twice = r * 2n;
  if (twice > d || (twice === d && q % 2n === 1n)) return q + 1n;
  return q;
}

function formatRate(scaled: bigint, decimals: number): string {
  const rounded = divRoundHalfEven(scaled, pow10(RATE_SCALE - decimals));
  return minorToDecimal(rounded, decimals);
}

/** to_minor = from_minor × rate, converting between the currencies' decimal places. */
export function convertFromTo(fromMinor: bigint, rateScaled: bigint, c: FxCurrencies): bigint {
  return divRoundHalfEven(
    fromMinor * rateScaled * pow10(c.toMinorUnits),
    SCALE * pow10(c.fromMinorUnits),
  );
}

/** from_minor = to_minor ÷ rate. */
export function convertToFrom(toMinor: bigint, rateScaled: bigint, c: FxCurrencies): bigint {
  return divRoundHalfEven(
    toMinor * SCALE * pow10(c.fromMinorUnits),
    rateScaled * pow10(c.toMinorUnits),
  );
}

/** rate = to ÷ from, scaled by 10^RATE_SCALE. */
export function deriveRate(fromMinor: bigint, toMinor: bigint, c: FxCurrencies): bigint {
  return divRoundHalfEven(
    toMinor * SCALE * pow10(c.fromMinorUnits),
    fromMinor * pow10(c.toMinorUnits),
  );
}

/** Applies a user edit to one field and recomputes the least recently touched other field. */
export function editFx(state: FxState, field: FxField, raw: string, c: FxCurrencies): FxState {
  const order = [...state.order.filter((f) => f !== field), field];
  const next: FxState = { ...state, order, [field]: raw };
  if (field === "rate") next.ratePrecise = raw;

  // The other two fields, oldest first; recompute the oldest.
  const target = order.find((f) => f !== field) as FxField;

  const from = parseAmount(next.from, c.fromMinorUnits);
  const to = parseAmount(next.to, c.toMinorUnits);
  const rate = parseRate(next.ratePrecise);

  switch (target) {
    case "to":
      next.to =
        from !== null && rate !== null
          ? minorToDecimal(convertFromTo(from, rate, c), c.toMinorUnits)
          : "";
      break;
    case "from":
      next.from =
        to !== null && rate !== null
          ? minorToDecimal(convertToFrom(to, rate, c), c.fromMinorUnits)
          : "";
      break;
    case "rate": {
      if (from !== null && to !== null && from > 0n && to > 0n) {
        const r = deriveRate(from, to, c);
        next.ratePrecise = formatRate(r, RATE_SCALE);
        next.rate = formatRate(r, RATE_DISPLAY);
      } else {
        next.rate = "";
        next.ratePrecise = "";
      }
      break;
    }
  }
  return next;
}

/** The values to submit, or null while the form is incomplete. */
export function fxResult(
  state: FxState,
  c: FxCurrencies,
): { fromMinor: bigint; toMinor: bigint; rate: string } | null {
  const from = parseAmount(state.from, c.fromMinorUnits);
  const to = parseAmount(state.to, c.toMinorUnits);
  const rate = parseRate(state.ratePrecise);
  if (from === null || to === null || rate === null || from <= 0n || to <= 0n) return null;
  return { fromMinor: from, toMinor: to, rate: formatRate(rate, RATE_SCALE) };
}
