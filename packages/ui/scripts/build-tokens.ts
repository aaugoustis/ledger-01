// Compiles tokens/tokens.json (the Module/01 design-system tokens, vendored from the artifact)
// into src/styles/tokens.generated.css: a Tailwind v4 @theme plus plain CSS variables.
// Run with `bun run tokens` after changing tokens.json.

import { join } from "node:path";

type Entry = { name: string; value: string | Record<string, string>; usage?: string };
type Family = { tokens: Entry[] };
type TextStyle = {
  name: string;
  fontSize: string;
  fontWeight: number | string;
  lineHeight: number | string;
  letterSpacing?: string;
};
export type Tokens = {
  name: string;
  color: { themes: { id: string }[]; tokens: Entry[] };
  type: { families: Record<string, string>; groups: { family: string; styles: TextStyle[] }[] };
  spacing: Family;
  radius: Family;
  notch: Family;
  shadow: Family;
  size: Family;
};

const value = (e: Entry): string =>
  typeof e.value === "string" ? e.value : (Object.values(e.value)[0] ?? "");

// A color alias "{other-token}" becomes a reference to that token's variable.
export const colorValue = (e: Entry): string =>
  value(e).replace(/^\{([A-Za-z0-9][A-Za-z0-9_.-]*)\}$/, "var(--color-$1)");

// "Michroma, \"Eurostile\", …" -> "var(--font-michroma, Michroma), \"Eurostile\", …" so next/font
// (which exposes hashed families through CSS variables) can override the first family.
export function fontStack(stack: string): string {
  const [first = "", ...rest] = stack.split(",").map((s) => s.trim());
  const bare = first.replace(/^["']|["']$/g, "");
  const slug = bare.toLowerCase().replace(/[^a-z0-9]+/g, "-");
  return [`var(--font-${slug}, ${first})`, ...rest].join(", ");
}

// Box-shadow lists can't be used with clip-path (it clips them), so notched parts use a chain
// of drop-shadow() filters instead. Inset shadows have no filter equivalent and are dropped.
export function dropShadowChain(boxShadow: string): string {
  return splitTopLevel(boxShadow)
    .filter((s) => !s.startsWith("inset"))
    .map((s) => `drop-shadow(${s})`)
    .join(" ");
}

function splitTopLevel(list: string): string[] {
  const out: string[] = [];
  let depth = 0;
  let cur = "";
  for (const ch of list) {
    if (ch === "(") depth++;
    if (ch === ")") depth--;
    if (ch === "," && depth === 0) {
      out.push(cur.trim());
      cur = "";
    } else cur += ch;
  }
  if (cur.trim()) out.push(cur.trim());
  return out;
}

export function generateTokensCss(t: Tokens): string {
  const theme: string[] = [];
  const root: string[] = [];

  // Colors: reset Tailwind's palette so only design-system colors exist.
  theme.push("  --color-*: initial;");
  for (const e of t.color.tokens) theme.push(`  --color-${e.name}: ${colorValue(e)};`);

  for (const [name, stack] of Object.entries(t.type.families)) {
    theme.push(`  --font-${name}: ${fontStack(stack)};`);
  }
  for (const group of t.type.groups) {
    for (const s of group.styles) {
      theme.push(`  --text-${s.name}: ${s.fontSize};`);
      theme.push(`  --text-${s.name}--line-height: ${s.lineHeight};`);
      theme.push(`  --text-${s.name}--font-weight: ${s.fontWeight};`);
      if (s.letterSpacing) theme.push(`  --text-${s.name}--letter-spacing: ${s.letterSpacing};`);
    }
  }

  // Module/01 spacing (4, 8, 12, 16, 24, 32, 48, 64, 96px) is exactly Tailwind's 4px scale, so the
  // default --spacing stays and `p-6` = space-6. Plain variables are still emitted for CSS use.
  for (const e of t.spacing.tokens) root.push(`  --${e.name}: ${value(e)};`);

  theme.push("  --radius-*: initial;");
  for (const e of t.radius.tokens) theme.push(`  --${e.name}: ${value(e)};`);

  theme.push("  --shadow-*: initial;");
  for (const e of t.shadow.tokens) {
    theme.push(`  --${e.name}: ${value(e)};`);
    root.push(`  --filter-${e.name.replace(/^shadow-/, "")}: ${dropShadowChain(value(e))};`);
  }

  for (const e of t.notch.tokens) root.push(`  --${e.name}: ${value(e)};`);

  for (const e of t.size.tokens) {
    const v = value(e);
    if (e.name.endsWith("-max")) theme.push(`  --container-${e.name}: ${v};`);
    else theme.push(`  --spacing-${e.name}: ${v};`);
  }

  return [
    `/* ${t.name} tokens — generated from tokens/tokens.json by scripts/build-tokens.ts. Do not edit. */`,
    "",
    "@theme static {",
    ...theme,
    "}",
    "",
    ":root {",
    ...root,
    "}",
    "",
  ].join("\n");
}

if (import.meta.main) {
  const dir = join(import.meta.dir, "..");
  const tokens = (await Bun.file(join(dir, "tokens/tokens.json")).json()) as Tokens;
  const out = join(dir, "src/styles/tokens.generated.css");
  await Bun.write(out, generateTokensCss(tokens));
  console.log(`wrote ${out}`);
}
