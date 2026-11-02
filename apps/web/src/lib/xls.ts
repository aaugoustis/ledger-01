// Legacy Excel (.xls) support for the bank import: the server reads CSV and XLSX only, so the
// browser converts .xls to .xlsx before uploading. SheetJS is loaded only when it's needed.

const XLSX_TYPE = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet";

/** Biggest .xls we'll try to convert; the converted file must still fit the upload cap. */
export const MAX_XLS_BYTES = 20 * 1024 * 1024;

/** Old .xls workbooks are OLE compound files, which start with D0 CF 11 E0. */
const OLE_SIGNATURE = [0xd0, 0xcf, 0x11, 0xe0];

/** True for a legacy Excel file, by its extension or its first bytes. */
export async function isLegacyExcel(file: File): Promise<boolean> {
  if (/\.xls$/i.test(file.name)) return true;
  if (/\.xlsx$/i.test(file.name)) return false;
  const head = new Uint8Array(await file.slice(0, 4).arrayBuffer());
  return OLE_SIGNATURE.every((b, i) => head[i] === b);
}

/**
 * Re-saves a legacy .xls workbook as .xlsx. Cells keep their raw values and number formats, so
 * dates stay date serials and the server reads the file exactly like one saved from Excel. Many
 * banks' ".xls" exports are really HTML tables; SheetJS reads those too.
 */
export async function xlsToXlsx(file: File): Promise<File> {
  const XLSX = await import("xlsx");
  const book = XLSX.read(new Uint8Array(await file.arrayBuffer()), {
    type: "array",
    cellNF: true,
    cellFormula: false,
    cellHTML: false,
  });
  const first = book.Sheets[book.SheetNames[0] ?? ""];
  const hasValues =
    first !== undefined &&
    Object.entries(first).some(([k, cell]) => !k.startsWith("!") && cell?.v !== undefined);
  if (!hasValues) throw new Error("The first sheet of this .xls file is empty");
  const out = XLSX.write(book, { bookType: "xlsx", type: "array", compression: true });
  const name = `${file.name.replace(/\.xls$/i, "")}.xlsx`;
  return new File([out], name, { type: XLSX_TYPE });
}
