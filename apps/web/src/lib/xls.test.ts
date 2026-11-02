import { describe, expect, test } from "bun:test";
import * as XLSX from "xlsx";
import { isLegacyExcel, xlsToXlsx } from "./xls";

// A made-up statement saved as legacy .xls (BIFF8), with a real date cell.
function legacyStatement(): File {
  const sheet = XLSX.utils.aoa_to_sheet([
    ["Date", "Description", "Amount"],
    [new Date(Date.UTC(2026, 8, 3)), "Corner Grocer", -45.2],
    [new Date(Date.UTC(2026, 8, 5)), "City Transit Card", -20],
  ]);
  const book = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(book, sheet, "Statement");
  const bytes = XLSX.write(book, { bookType: "biff8", type: "array" });
  return new File([bytes], "northbank.xls", { type: "application/vnd.ms-excel" });
}

describe("legacy Excel", () => {
  test("detects .xls by extension or signature", async () => {
    const xls = legacyStatement();
    expect(await isLegacyExcel(xls)).toBe(true);
    // Renamed, it's still recognised by its OLE signature.
    expect(await isLegacyExcel(new File([xls], "statement.bin"))).toBe(true);
    expect(await isLegacyExcel(new File(["Date,Amount\n"], "statement.csv"))).toBe(false);
    expect(await isLegacyExcel(new File([xls], "statement.xlsx"))).toBe(false);
  });

  test("converts to .xlsx keeping raw values and date serials", async () => {
    const out = await xlsToXlsx(legacyStatement());
    expect(out.name).toBe("northbank.xlsx");
    const head = new Uint8Array(await out.slice(0, 2).arrayBuffer());
    expect([...head]).toEqual([0x50, 0x4b]); // a zip, like every .xlsx

    const book = XLSX.read(new Uint8Array(await out.arrayBuffer()), { type: "array" });
    const sheet = book.Sheets[book.SheetNames[0] ?? ""];
    if (!sheet) throw new Error("no sheet");
    expect(sheet.B2?.v).toBe("Corner Grocer");
    expect(sheet.C2?.v).toBe(-45.2);
    // 2026-09-03 is serial 46268 in the 1900 date system.
    expect(sheet.A2?.t).toBe("n");
    expect(Math.round(Number(sheet.A2?.v))).toBe(46268);
  });

  test("converts an HTML table saved as .xls, as some banks export", async () => {
    const html =
      "<table><tr><td>Date</td><td>Amount</td></tr><tr><td>2026-09-03</td><td>-45.20</td></tr></table>";
    const out = await xlsToXlsx(new File([html], "export.xls"));
    const book = XLSX.read(new Uint8Array(await out.arrayBuffer()), { type: "array" });
    const sheet = book.Sheets[book.SheetNames[0] ?? ""];
    expect(sheet?.A1?.v).toBe("Date");
  });

  test("rejects a workbook with an empty first sheet", async () => {
    const book = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(book, XLSX.utils.aoa_to_sheet([]), "Empty");
    const bytes = XLSX.write(book, { bookType: "biff8", type: "array" });
    await expect(xlsToXlsx(new File([bytes], "empty.xls"))).rejects.toThrow("empty");
  });
});
