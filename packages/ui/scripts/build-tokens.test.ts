import { describe, expect, test } from "bun:test";
import tokens from "../tokens/tokens.json";
import { dropShadowChain, fontStack, generateTokensCss, type Tokens } from "./build-tokens";

describe("fontStack", () => {
  test("wraps the first family in a next/font variable", () => {
    expect(fontStack('Michroma, "Eurostile", sans-serif')).toBe(
      'var(--font-michroma, Michroma), "Eurostile", sans-serif',
    );
    expect(fontStack('"Space Mono", monospace')).toBe(
      'var(--font-space-mono, "Space Mono"), monospace',
    );
  });
});

describe("dropShadowChain", () => {
  test("converts a box-shadow list and drops insets", () => {
    expect(
      dropShadowChain("inset 0 1px 0 rgba(255,255,255,0.9), 0 2px 4px rgba(28,30,33,0.06)"),
    ).toBe("drop-shadow(0 2px 4px rgba(28,30,33,0.06))");
  });
});

describe("generateTokensCss", () => {
  const css = generateTokensCss(tokens as unknown as Tokens);

  test("emits every color token into the theme", () => {
    for (const t of (tokens as unknown as Tokens).color.tokens) {
      expect(css).toContain(`--color-${t.name}: `);
    }
    expect(css).toContain("--color-signal: #ff5f1f;");
    // Aliases resolve to the referenced token's variable.
    expect(css).toContain("--color-chart-line: var(--color-signal-ink);");
    expect(css).not.toMatch(/:\s*\{[a-z]/);
    expect(css).toContain("--color-*: initial;");
  });

  test("emits text styles with sub-properties", () => {
    expect(css).toContain("--text-display-xl: 56px;");
    expect(css).toContain("--text-hud--letter-spacing: 0.12em;");
    expect(css).toContain("--text-amount-md--font-weight: 700;");
  });

  test("emits radius, notch, shadows and sizes", () => {
    expect(css).toContain("--radius-lg: 24px;");
    expect(css).toContain("--notch-md: 20px;");
    expect(css).toContain("--filter-float: drop-shadow(");
    expect(css).toContain("--spacing-control-height: 44px;");
    expect(css).toContain("--container-content-max: 1120px;");
  });
});
