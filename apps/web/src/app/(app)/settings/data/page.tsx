"use client";

import { useMutation } from "@connectrpc/connect-query";
import { BackupService } from "@ledger-01/api";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@ledger-01/ui/components/alert-dialog";
import { Button } from "@ledger-01/ui/components/button";
import { DatePicker } from "@ledger-01/ui/components/date-picker";
import { Field } from "@ledger-01/ui/components/field";
import { Input } from "@ledger-01/ui/components/input";
import { MultiCombobox } from "@ledger-01/ui/components/multi-combobox";
import { Panel } from "@ledger-01/ui/components/panel";
import { Download, FileSpreadsheet, Upload } from "lucide-react";
import { useRouter } from "next/navigation";
import { useId, useRef, useState } from "react";
import { toast } from "sonner";
import { useHousehold, useWallets } from "@/lib/api/hooks";
import { type BackupSummary, readBackupSummary } from "@/lib/backup";
import { today } from "@/lib/period";

const dateFormat = new Intl.DateTimeFormat("en", { dateStyle: "medium", timeStyle: "short" });
const numberFormat = new Intl.NumberFormat("en");

function save(content: Uint8Array, fileName: string, type: string) {
  const url = URL.createObjectURL(new Blob([content as BlobPart], { type }));
  const a = document.createElement("a");
  a.href = url;
  a.download = fileName;
  a.click();
  URL.revokeObjectURL(url);
}

/** Backups, restore and CSV export of the household's ledger. */
export default function DataPage() {
  const id = useId();
  const router = useRouter();
  const { wallets } = useWallets();
  const exportBackup = useMutation(BackupService.method.exportBackup);
  const exportCsv = useMutation(BackupService.method.exportTransactionsCsv);
  const restore = useMutation(BackupService.method.restoreBackup);
  const { isOwner } = useHousehold();

  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [walletIds, setWalletIds] = useState<string[]>([]);

  const fileInput = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<{ content: Uint8Array; summary: BackupSummary }>();
  const [fileError, setFileError] = useState<string>();
  const [confirming, setConfirming] = useState(false);

  const choose = async (f: File | undefined) => {
    setFile(undefined);
    setFileError(undefined);
    if (!f) return;
    if (f.size > 5 * 1024 * 1024) {
      setFileError("The file is larger than 5 MB.");
      return;
    }
    try {
      const content = new Uint8Array(await f.arrayBuffer());
      setFile({ content, summary: await readBackupSummary(content) });
    } catch (err) {
      setFileError(err instanceof Error ? err.message : "This isn't a Ledger/01 backup.");
    }
  };

  const counts = file?.summary.counts;

  return (
    <div className="flex flex-col gap-6">
      <Panel serial="DAT-01 · BACKUP" className="max-w-2xl">
        <div className="flex flex-col gap-3">
          <p className="text-body-sm text-ink">
            Download everything in this household as one file: wallets, categories, labels,
            transactions, budgets, recurring transactions and import settings. Restore it here or on
            another Ledger/01.
          </p>
          <Button
            className="self-start"
            disabled={exportBackup.isPending}
            onClick={async () => {
              const res = await exportBackup.mutateAsync({});
              save(res.content, res.fileName, "application/gzip");
            }}
          >
            <Download strokeWidth={1.25} /> Download backup
          </Button>
        </div>
      </Panel>

      <Panel serial="DAT-02 · CSV" className="max-w-2xl">
        <div className="flex flex-col gap-4">
          <p className="text-body-sm text-ink">
            Transactions as a spreadsheet, oldest first. Amounts are in each wallet's currency, with
            money going out as negative. Upcoming transactions are left out.
          </p>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="From" htmlFor={`${id}-from`}>
              <DatePicker
                id={`${id}-from`}
                value={from}
                onChange={setFrom}
                placeholder="First one"
              />
            </Field>
            <Field label="To" htmlFor={`${id}-to`}>
              <DatePicker id={`${id}-to`} value={to} onChange={setTo} placeholder="Today" />
            </Field>
          </div>
          <Field label="Wallets" htmlFor={`${id}-wallets`}>
            <MultiCombobox
              id={`${id}-wallets`}
              value={walletIds}
              onChange={setWalletIds}
              options={wallets}
              getKey={(w) => w.id}
              getSearchText={(w) => `${w.name} ${w.currency}`}
              renderOption={(w) => (
                <>
                  <span className="truncate">{w.name}</span>
                  <span className="hud-sm text-ink-soft">{w.currency}</span>
                </>
              )}
              allLabel="All wallets"
              noun="wallets"
            />
          </Field>
          <Button
            variant="secondary"
            className="self-start"
            disabled={exportCsv.isPending}
            onClick={async () => {
              const res = await exportCsv.mutateAsync({
                today: today(),
                from: from || undefined,
                to: to || undefined,
                walletIds,
              });
              if (res.rowCount === 0) {
                toast("No transactions to export");
                return;
              }
              save(res.content, res.fileName, "text/csv");
            }}
          >
            <FileSpreadsheet strokeWidth={1.25} /> Export transactions
          </Button>
        </div>
      </Panel>

      {isOwner ? (
        <Panel serial="DAT-03 · RESTORE" className="max-w-2xl">
          <div className="flex flex-col gap-4">
            <p className="text-body-sm text-ink">
              Restoring a backup replaces everything in this household with what's in the file.
            </p>
            <Field label="Backup file" htmlFor={`${id}-file`} error={fileError}>
              <Input
                ref={fileInput}
                id={`${id}-file`}
                type="file"
                accept=".gz,.json,application/gzip,application/json"
                disabled={restore.isPending}
                onChange={(e) => choose(e.target.files?.[0])}
                className="pt-2.5"
              />
            </Field>
            {file && counts && (
              <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-body-sm sm:grid-cols-4">
                <dt className="col-span-2 text-ink-soft sm:col-span-4">
                  {file.summary.household ? `${file.summary.household} · ` : ""}
                  {file.summary.exportedAt
                    ? `saved ${dateFormat.format(file.summary.exportedAt)}`
                    : "date unknown"}
                </dt>
                {(
                  [
                    ["Wallets", counts.wallets],
                    ["Transactions", counts.transactions],
                    ["Transfers", counts.transfers],
                    ["Categories", counts.categories],
                    ["Labels", counts.labels],
                    ["Budgets", counts.budgets],
                    ["Recurring", counts.recurringRules],
                    ["Import rules", counts.importRules],
                  ] as const
                ).map(([label, n]) => (
                  <div
                    key={label}
                    className="flex justify-between gap-2 sm:flex-col sm:justify-start"
                  >
                    <dt className="text-ink-soft">{label}</dt>
                    <dd className="text-ink tabular-nums">{numberFormat.format(n)}</dd>
                  </div>
                ))}
              </dl>
            )}
            <Button
              variant="destructive"
              className="self-start"
              disabled={!file || restore.isPending}
              onClick={() => setConfirming(true)}
            >
              <Upload strokeWidth={1.25} /> {restore.isPending ? "Restoring…" : "Restore backup"}
            </Button>
          </div>
        </Panel>
      ) : (
        <Panel serial="DAT-03 · RESTORE" recess className="max-w-2xl">
          <p className="text-body-sm text-ink">Only the household owner can restore a backup.</p>
        </Panel>
      )}

      <AlertDialog open={confirming} onOpenChange={setConfirming}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Replace all data in this household?</AlertDialogTitle>
            <AlertDialogDescription>
              Every wallet, transaction, category, label, budget, recurring transaction and import
              rule here is deleted and replaced with the backup's. This can't be undone, so download
              a backup first if you might need the current data.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={async () => {
                if (!file) return;
                const res = await restore.mutateAsync({ content: file.content, replace: true });
                const n = res.counts?.transactions ?? 0;
                toast.success(`Restored ${numberFormat.format(n)} transactions`);
                setFile(undefined);
                if (fileInput.current) fileInput.current.value = "";
                router.refresh();
              }}
            >
              Replace data
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
