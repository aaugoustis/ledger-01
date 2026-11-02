import { describe, expect, test } from "bun:test";
import { formatMoney, minorToDecimal, parseAmount } from "./money";

describe("minorToDecimal", () => {
  test("handles scale, sign and zero decimals", () => {
    expect(minorToDecimal(123456n, 2)).toBe("1234.56");
    expect(minorToDecimal(-5n, 2)).toBe("-0.05");
    expect(minorToDecimal(1866n, 0)).toBe("1866");
  });
});

describe("parseAmount", () => {
  test("accepts plain and grouped decimals", () => {
    expect(parseAmount("1,234.5", 2)).toBe(123450n);
    expect(parseAmount("0.99", 2)).toBe(99n);
    expect(parseAmount(".5", 2)).toBe(50n);
    expect(parseAmount("42", 0)).toBe(42n);
  });

  test("rejects too many decimals, signs and junk", () => {
    expect(parseAmount("1.234", 2)).toBeNull();
    expect(parseAmount("-1", 2)).toBeNull();
    expect(parseAmount("1e3", 2)).toBeNull();
    expect(parseAmount("", 2)).toBeNull();
  });
});

describe("formatMoney", () => {
  test("formats like the app", () => {
    expect(formatMoney(123456n, "USD", 2, { sign: "always" })).toBe("+1,234.56 USD");
    expect(formatMoney(-4205n, "USD", 2)).toBe("−42.05 USD");
    expect(formatMoney(0n, "USD", 2, { sign: "always" })).toBe("0.00 USD");
    expect(formatMoney(-470000n, "UYU", 2, { code: false })).toBe("−4,700.00");
  });

  test("keeps precision for large amounts", () => {
    expect(formatMoney(900719925474099312n, "USD", 2)).toBe("9,007,199,254,740,993.12 USD");
  });
});
