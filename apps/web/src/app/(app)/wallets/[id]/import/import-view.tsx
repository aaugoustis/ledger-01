"use client";

import { useMutation, useQuery } from "@connectrpc/connect-query";
import {
  type ImportMapping,
  ImportService,
  type UploadImportFileResponse,
  WalletService,
} from "@ledger-01/api";
import { Button } from "@ledger-01/ui/components/button";
import { Field } from "@ledger-01/ui/components/field";
import { HudLabel } from "@ledger-01/ui/components/hud";
import { Input } from "@ledger-01/ui/components/input";
import { Panel } from "@ledger-01/ui/components/panel";
import { fault } from "@ledger-01/ui/lib/fault";
import { CircleCheck, FileUp } from "lucide-react";
import Link from "next/link";
import { useId, useState } from "react";
import { toast } from "sonner";
import { MAX_IMPORT_BYTES } from "@/lib/imports";
import { today } from "@/lib/period";
import { useFileDrop } from "@/lib/use-file-drop";
import { isLegacyExcel, MAX_XLS_BYTES, xlsToXlsx } from "@/lib/xls";
import { MappingStep } from "./mapping-step";
import { ReviewStep } from "./review-step";

type Step = "file" | "columns" | "review" | "done";

const stepTitles: Record<Step, string> = {
  file: "Choose a file",
  columns: "Match the columns",
  review: "Review the rows",
  done: "Imported",
};

/**
 * Bank statement import into one wallet: file → columns → review → import. Each step reads the
 * staged upload again on the server, so nothing but the mapping lives in the browser. A wallet
 * whose columns were saved last time goes straight from the file to the review. A file can be
 * dropped anywhere on the page at any step (a later step starts over with it), or chosen with the
 * file input.
 */
export function ImportView({ walletId }: { walletId: string }) {
  const id = useId();
  const walletQ = useQuery(WalletService.method.getWallet, { id: walletId, today: today() });
  const wallet = walletQ.data?.wallet;
  const [step, setStep] = useState<Step>("file");
  const [upload, setUpload] = useState<UploadImportFileResponse | null>(null);
  const [mapping, setMapping] = useState<ImportMapping | null>(null);
  const [imported, setImported] = useState(0n);
  const [fileError, setFileError] = useState<string>();
  const [converting, setConverting] = useState(false);
  const send = useMutation(ImportService.method.uploadImportFile);
  const busy = send.isPending || converting;
  const dragging = useFileDrop(Boolean(wallet) && !busy, (file) => choose(file));

  function rejectFile(message: string) {
    setFileError(message);
    toast.error(message);
    fault(`${id}-file`);
  }

  async function choose(picked: File | undefined) {
    setFileError(undefined);
    if (!picked) return;
    let file = picked;
    // The server reads CSV and XLSX; old .xls workbooks are converted here first.
    if (await isLegacyExcel(file)) {
      if (file.size > MAX_XLS_BYTES) return rejectFile("Choose an .xls file of 20 MB or less");
      setConverting(true);
      try {
        file = await xlsToXlsx(file);
      } catch {
        return rejectFile("This .xls file can't be read. Save it as .xlsx or .csv and try again");
      } finally {
        setConverting(false);
      }
    }
    if (file.size > MAX_IMPORT_BYTES) return rejectFile("Choose a file of 4 MB or less");
    try {
      const res = await send.mutateAsync({
        walletId,
        fileName: file.name,
        content: new Uint8Array(await file.arrayBuffer()),
      });
      setUpload(res);
      setMapping(res.mapping ?? null);
      setStep(res.savedMapping ? "review" : "columns");
    } catch {
      // The mutation toasted the reason (a damaged file, too many rows, …).
      fault(`${id}-file`);
    }
  }

  if (walletQ.isError) {
    return (
      <Panel recess className="max-w-xl">
        <p className="text-body-md text-ink">This wallet doesn't exist or isn't yours.</p>
        <Link
          href="/wallets"
          className="mt-4 inline-block text-signal-ink underline-offset-4 hover:underline"
        >
          Back to wallets
        </Link>
      </Panel>
    );
  }

  return (
    <div className="flex flex-col gap-8">
      <div className="flex flex-col gap-3">
        <HudLabel code="S-02" end={wallet?.currency}>
          <Link href={`/wallets/${walletId}`} className="hover:text-ink">
            {wallet?.name ?? "Wallet"}
          </Link>
        </HudLabel>
        <h1 className="font-display text-display-md text-ink sm:text-display-lg">Import</h1>
        <ol className="hud-sm flex flex-wrap gap-x-4 gap-y-1 text-ink-soft" aria-label="Steps">
          {(["file", "columns", "review"] as const).map((s, i) => (
            <li
              key={s}
              aria-current={s === step ? "step" : undefined}
              className="aria-current:text-ink"
            >
              {i + 1}. {stepTitles[s]}
            </li>
          ))}
        </ol>
      </div>

      {step === "file" && (
        <Panel className="max-w-2xl">
          <div className="flex flex-col gap-4">
            <p className="text-body-md text-ink-soft">
              Export your bank statement as CSV, XLSX or XLS (up to 4 MB) and drop it anywhere on
              this page, or choose it below. Amounts are imported in{" "}
              {wallet?.currency ?? "the wallet's currency"}.
            </p>
            <Field label="Statement file" htmlFor={`${id}-file`} error={fileError}>
              <Input
                id={`${id}-file`}
                type="file"
                accept=".csv,.tsv,.txt,.xlsx,.xls,text/csv,text/plain,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/vnd.ms-excel"
                disabled={send.isPending || converting || !wallet}
                onChange={(e) => choose(e.target.files?.[0])}
                className="pt-2.5"
              />
            </Field>
            {converting && <p className="text-body-sm text-ink-soft">Converting the .xls file…</p>}
            {send.isPending && <p className="text-body-sm text-ink-soft">Reading the file…</p>}
          </div>
        </Panel>
      )}

      {step === "columns" && upload && mapping && wallet && (
        <MappingStep
          upload={upload}
          mapping={mapping}
          onChange={setMapping}
          currency={wallet.currency}
          onBack={() => setStep("file")}
          onNext={() => setStep("review")}
        />
      )}

      {step === "review" && upload && mapping && wallet && (
        <ReviewStep
          batchId={upload.batchId}
          mapping={mapping}
          wallet={wallet}
          savedMapping={upload.savedMapping}
          onBack={() => setStep("columns")}
          onDone={(n) => {
            setImported(n);
            setStep("done");
            toast.success(`Imported ${n} transactions`);
          }}
        />
      )}

      {step === "done" && (
        <Panel className="max-w-xl">
          <div className="flex flex-col items-start gap-4">
            <p className="flex items-center gap-2 text-body-md text-ink">
              <CircleCheck strokeWidth={1.25} className="size-5 text-income-ink" />
              Imported {imported.toString()} transactions into {wallet?.name}.
            </p>
            <div className="flex flex-wrap gap-2">
              <Button asChild>
                <Link href={`/wallets/${walletId}`}>Back to the wallet</Link>
              </Button>
              <Button
                variant="secondary"
                onClick={() => {
                  setUpload(null);
                  setStep("file");
                }}
              >
                Import another file
              </Button>
            </div>
          </div>
        </Panel>
      )}

      {dragging && (
        <div
          aria-hidden
          className="pointer-events-none fixed inset-0 z-50 flex items-center justify-center bg-overlay p-6"
        >
          <div className="flex flex-col items-center gap-3 rounded-lg border-2 border-dashed border-signal bg-shell px-10 py-8 text-center shadow-float">
            <FileUp strokeWidth={1.25} className="size-8 text-signal-ink" />
            <p className="font-display text-display-sm text-ink">Drop the statement</p>
            <p className="text-body-sm text-ink-soft">
              {step === "file"
                ? "CSV, XLSX or XLS, up to 4 MB"
                : "It replaces the file you're importing now"}
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
