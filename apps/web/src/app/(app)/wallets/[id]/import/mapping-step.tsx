"use client";

import { useQuery } from "@connectrpc/connect-query";
import {
  DecimalSeparator,
  type ImportBank,
  type ImportMapping,
  ImportService,
  ImportSign,
  TransactionKind,
  type UploadImportFileResponse,
} from "@ledger-01/api";
import { Amount } from "@ledger-01/ui/components/amount";
import { Button } from "@ledger-01/ui/components/button";
import { Checkbox } from "@ledger-01/ui/components/checkbox";
import { Field } from "@ledger-01/ui/components/field";
import { Label } from "@ledger-01/ui/components/label";
import { Panel } from "@ledger-01/ui/components/panel";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@ledger-01/ui/components/select";
import { Skeleton } from "@ledger-01/ui/components/skeleton";
import { fault } from "@ledger-01/ui/lib/fault";
import { useId, useState } from "react";
import { useCurrencies } from "@/lib/api/hooks";
import {
  type AmountMode,
  amountMode,
  checkMapping,
  columnFromValue,
  columnOptions,
  columnValue,
  formatExample,
  NONE,
  tableStartOptions,
} from "@/lib/imports";
import { formatDay } from "@/lib/period";

const PREVIEW_ROWS = 6;

const regions = new Intl.DisplayNames(["en"], { type: "region" });

/** "Itaú · Uruguay", or just the name for banks without a country ("Other bank"). */
function bankLabel(b: ImportBank) {
  return b.country ? `${b.name} · ${regions.of(b.country) ?? b.country}` : b.name;
}

type Option = { value: string; label: string };

function Choice({
  id,
  label,
  value,
  options,
  onChange,
  error,
  help,
}: {
  id: string;
  label: string;
  value: string;
  options: Option[];
  onChange: (v: string) => void;
  error?: string;
  help?: string;
}) {
  return (
    <Field label={label} htmlFor={id} error={error} help={help}>
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger id={id} className="w-full min-w-0">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {options.map((o) => (
            <SelectItem key={o.value} value={o.value}>
              {o.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </Field>
  );
}

/** Which file column holds what, with a live preview of the first rows read that way. */
export function MappingStep({
  upload,
  mapping,
  onChange,
  currency,
  onBack,
  onNext,
}: {
  upload: UploadImportFileResponse;
  mapping: ImportMapping;
  onChange: (m: ImportMapping) => void;
  currency: string;
  onBack: () => void;
  onNext: () => void;
}) {
  const id = useId();
  const { minorUnits } = useCurrencies();
  const [mode, setMode] = useState<AmountMode>(amountMode(mapping));
  const [errors, setErrors] = useState<{ amount?: string; debit?: string }>({});
  const sample = upload.sample.map((r) => r.cells);
  const columns = columnOptions(sample.slice(mapping.skipRows), mapping.hasHeader);
  const starts = tableStartOptions(sample);
  const bankOptions = upload.banks.map((b) => ({ value: b.id, label: bankLabel(b) }));
  const optional = [{ value: NONE, label: "None" }, ...columns];
  const set = (patch: Partial<ImportMapping>) => {
    setErrors({});
    onChange({ ...mapping, ...patch });
  };
  const fault_ = checkMapping(mapping, mode);

  const preview = useQuery(
    ImportService.method.previewImport,
    { batchId: upload.batchId, mapping, limit: PREVIEW_ROWS },
    { enabled: !fault_, retry: false },
  );

  function next() {
    if (fault_) {
      setErrors({ [fault_.field]: fault_.message });
      fault(`${id}-${fault_.field}`);
      return;
    }
    onNext();
  }

  return (
    <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
      <Panel>
        <form
          className="flex flex-col gap-5"
          onSubmit={(e) => {
            e.preventDefault();
            next();
          }}
        >
          <p className="text-body-sm text-ink-soft">
            {upload.rowCount} rows, {upload.columnCount} columns
            {upload.format === "csv" ? ` · ${upload.encoding}` : " · XLSX"}
          </p>
          {/* Banks often put a logo, the account holder and other details above the table. */}
          <Choice
            id={`${id}-start`}
            label="The table starts at"
            value={String(mapping.skipRows)}
            options={starts}
            onChange={(v) => set({ skipRows: Number(v) })}
          />
          <div className="flex items-center gap-2">
            <Checkbox
              id={`${id}-header`}
              checked={mapping.hasHeader}
              onCheckedChange={(v) => set({ hasHeader: v === true })}
            />
            <Label htmlFor={`${id}-header`} className="text-body-sm text-ink">
              That row has the column names
            </Label>
          </div>
          <Choice
            id={`${id}-bank`}
            label="Bank"
            value={mapping.bank || "generic"}
            options={bankOptions}
            help="Used to recognise discounts and refunds in the descriptions."
            onChange={(v) => set({ bank: v })}
          />

          <div className="grid gap-4 sm:grid-cols-2">
            <Choice
              id={`${id}-date`}
              label="Date column"
              value={String(mapping.dateColumn)}
              options={columns}
              onChange={(v) => set({ dateColumn: Number(v) })}
            />
            <Choice
              id={`${id}-format`}
              label="Date format"
              value={mapping.dateFormat}
              options={upload.dateFormats.map((f) => ({
                value: f,
                label: `${f} (${formatExample(f)})`,
              }))}
              onChange={(v) => set({ dateFormat: v })}
            />
          </div>

          <Choice
            id={`${id}-mode`}
            label="Amounts"
            value={mode}
            options={[
              { value: "single", label: "One column, signed" },
              { value: "split", label: "Money out and money in columns" },
            ]}
            onChange={(v) => {
              setMode(v as AmountMode);
              set(
                v === "split"
                  ? { amountColumn: undefined }
                  : { debitColumn: undefined, creditColumn: undefined },
              );
            }}
          />
          {mode === "single" ? (
            <div className="grid gap-4 sm:grid-cols-2">
              <Choice
                id={`${id}-amount`}
                label="Amount column"
                value={columnValue(mapping.amountColumn)}
                options={optional}
                error={errors.amount}
                onChange={(v) => set({ amountColumn: columnFromValue(v) })}
              />
              <Choice
                id={`${id}-sign`}
                label="Expenses are"
                value={String(mapping.sign || ImportSign.NEGATIVE_IS_EXPENSE)}
                options={[
                  { value: String(ImportSign.NEGATIVE_IS_EXPENSE), label: "Negative amounts" },
                  { value: String(ImportSign.POSITIVE_IS_EXPENSE), label: "Positive amounts" },
                ]}
                onChange={(v) => set({ sign: Number(v) as ImportSign })}
              />
            </div>
          ) : (
            <div className="grid gap-4 sm:grid-cols-2">
              <Choice
                id={`${id}-debit`}
                label="Money out column"
                value={columnValue(mapping.debitColumn)}
                options={optional}
                error={errors.debit}
                onChange={(v) => set({ debitColumn: columnFromValue(v) })}
              />
              <Choice
                id={`${id}-credit`}
                label="Money in column"
                value={columnValue(mapping.creditColumn)}
                options={optional}
                onChange={(v) => set({ creditColumn: columnFromValue(v) })}
              />
            </div>
          )}
          <Choice
            id={`${id}-decimal`}
            label="Decimal separator"
            value={String(mapping.decimalSeparator || DecimalSeparator.DOT)}
            options={[
              { value: String(DecimalSeparator.DOT), label: "Dot: 1,234.56" },
              { value: String(DecimalSeparator.COMMA), label: "Comma: 1.234,56" },
            ]}
            onChange={(v) => set({ decimalSeparator: Number(v) as DecimalSeparator })}
          />

          <div className="grid gap-4 sm:grid-cols-2">
            <Choice
              id={`${id}-note`}
              label="Note column (optional)"
              value={columnValue(mapping.noteColumn)}
              options={optional}
              onChange={(v) => set({ noteColumn: columnFromValue(v) })}
            />
            <Choice
              id={`${id}-category`}
              label="Category column (optional)"
              value={columnValue(mapping.categoryColumn)}
              options={optional}
              onChange={(v) => set({ categoryColumn: columnFromValue(v) })}
            />
          </div>

          <div className="flex flex-wrap justify-between gap-2 pt-2">
            <Button type="button" variant="ghost" onClick={onBack}>
              Choose another file
            </Button>
            <Button type="submit">Review rows</Button>
          </div>
        </form>
      </Panel>

      <Panel serial="PREVIEW" notch="sm">
        <div className="flex flex-col gap-3">
          <h2 className="text-label text-ink">First rows, read this way</h2>
          {fault_ ? (
            <p className="text-body-sm text-ink-soft">{fault_.message} to see a preview.</p>
          ) : preview.isPending ? (
            <div className="flex flex-col gap-2">
              {[0, 1, 2].map((i) => (
                <Skeleton key={i} className="h-9" />
              ))}
            </div>
          ) : preview.isError ? (
            <p className="text-body-sm text-danger">{preview.error.rawMessage}</p>
          ) : (
            <ul className="flex flex-col">
              {preview.data?.rows.map((r) => (
                <li
                  key={r.index}
                  className="flex items-center gap-3 border-b border-seam/60 py-2 last:border-b-0"
                >
                  {r.error ? (
                    <span className="text-body-sm text-danger">
                      Row {r.index + 1}: {r.error}
                    </span>
                  ) : (
                    <>
                      <span className="flex min-w-0 flex-1 flex-col">
                        <span className="truncate text-body-sm text-ink">{r.note || "—"}</span>
                        <span className="hud-sm text-ink-soft">{formatDay(r.occurredOn)}</span>
                      </span>
                      <Amount
                        minor={r.kind === TransactionKind.EXPENSE ? -r.amountMinor : r.amountMinor}
                        currency={currency}
                        minorUnits={minorUnits(currency)}
                        size="sm"
                      />
                    </>
                  )}
                </li>
              ))}
            </ul>
          )}
        </div>
      </Panel>
    </div>
  );
}
