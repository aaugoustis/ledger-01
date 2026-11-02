// Renders a docs SVG (docs/brand/*.svg) to PNG with the fonts it imports, for the README and docs.
// GitHub doesn't load web fonts inside SVG images, so the docs embed the PNG.
// Usage: bun scripts/render-svg.ts <in.svg> <out.png> <width>
import { readFileSync } from "node:fs";
import { chromium } from "@playwright/test";

const [src, out, widthArg] = process.argv.slice(2);
if (!src || !out || !widthArg) throw new Error("usage: render-svg.ts <in.svg> <out.png> <width>");

const svg = readFileSync(src, "utf8");
const [, vw, vh] = svg.match(/viewBox="0 0 (\d+) (\d+)"/) ?? [];
const width = Number(widthArg);
const height = Math.round((width * Number(vh)) / Number(vw));

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width, height } });
const sized = svg.replace(/width="\d+" height="\d+"/, `width="${width}" height="${height}"`);
await page.setContent(`<html><body style="margin:0;background:transparent">${sized}</body></html>`);
await page.evaluate(async () => {
  await Promise.all(
    ["30px Michroma", "11px 'Space Mono'", "bold 11px 'Space Mono'", "bold 17px Manrope"].map((f) =>
      document.fonts.load(f),
    ),
  );
  await document.fonts.ready;
});
await page.locator("svg").screenshot({ path: out, omitBackground: true });
await browser.close();
