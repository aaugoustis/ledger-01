"use client";

import { useMutation, useQuery } from "@connectrpc/connect-query";
import { CategoryKind, CategoryService, WalletService } from "@ledger-01/api";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@ledger-01/ui/components/alert-dialog";
import { Button } from "@ledger-01/ui/components/button";
import { CategoryIcon } from "@ledger-01/ui/components/category-icon";
import { Checkbox } from "@ledger-01/ui/components/checkbox";
import { Field } from "@ledger-01/ui/components/field";
import { HudLabel, SectionHeader } from "@ledger-01/ui/components/hud";
import { Input } from "@ledger-01/ui/components/input";
import { Label } from "@ledger-01/ui/components/label";
import { Panel } from "@ledger-01/ui/components/panel";
import { Eye, EyeOff } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { useCategories } from "@/lib/api/hooks";
import { today } from "@/lib/period";

export function WalletSettings({ walletId }: { walletId: string }) {
  const router = useRouter();
  const { data } = useQuery(WalletService.method.getWallet, {
    id: walletId,
    today: today(),
  });
  const wallet = data?.wallet;
  const { categories } = useCategories();
  const [name, setName] = useState("");
  useEffect(() => setName(wallet?.name ?? ""), [wallet?.name]);

  const update = useMutation(WalletService.method.updateWallet);
  const archive = useMutation(WalletService.method.archiveWallet);
  const remove = useMutation(WalletService.method.deleteWallet);
  const setHidden = useMutation(CategoryService.method.setCategoryHidden);

  if (!wallet) return null;
  const hasTransactions = wallet.transactionCount > 0n;

  return (
    <div className="flex flex-col gap-10">
      <SectionHeader
        code="S-03"
        label={
          <Link href={`/wallets/${walletId}`} className="hover:text-ink">
            {wallet.name}
          </Link>
        }
        title="Wallet settings"
      />

      <Panel serial="CFG-01 · GENERAL" className="max-w-2xl">
        <form
          className="flex flex-col gap-5"
          onSubmit={async (e) => {
            e.preventDefault();
            await update.mutateAsync({ id: walletId, name });
            toast.success("Saved");
          }}
        >
          <Field label="Wallet name" htmlFor="wallet-name">
            <Input
              id="wallet-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={60}
              required
            />
          </Field>
          <div className="flex items-center gap-2">
            <Checkbox
              id="wallet-hidden"
              checked={wallet.hidden}
              onCheckedChange={(v) => update.mutate({ id: walletId, hidden: v === true })}
            />
            <Label htmlFor="wallet-hidden" className="text-body-sm">
              Hide this wallet from the wallet list and totals
            </Label>
          </div>
          <Button
            type="submit"
            className="self-start"
            disabled={update.isPending || name.trim() === wallet.name}
          >
            Save
          </Button>
        </form>
      </Panel>

      <Panel serial="CFG-02 · CATEGORIES" className="max-w-2xl">
        <div className="flex flex-col gap-4">
          <p className="text-body-sm text-ink-soft">
            Hidden categories don't show up when you add transactions to this wallet. Existing
            transactions keep them.
          </p>
          {[CategoryKind.EXPENSE, CategoryKind.INCOME].map((kind) => (
            <div key={kind} className="flex flex-col gap-1">
              <HudLabel rule>{kind === CategoryKind.EXPENSE ? "Expense" : "Income"}</HudLabel>
              <ul className="grid gap-1 sm:grid-cols-2">
                {categories
                  .filter((c) => c.kind === kind && !c.archived)
                  .map((c) => {
                    const hidden = c.hiddenInWalletIds.includes(walletId);
                    return (
                      <li
                        key={c.id}
                        className="flex items-center gap-3 rounded-md px-2 py-1.5 hover:bg-shell-recess"
                      >
                        <CategoryIcon icon={c.icon} color={c.color} size="sm" />
                        <span
                          className={
                            hidden
                              ? "flex-1 text-body-sm text-ink-soft line-through"
                              : "flex-1 text-body-sm"
                          }
                        >
                          {c.name}
                        </span>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          aria-label={hidden ? `Show ${c.name}` : `Hide ${c.name}`}
                          onClick={() =>
                            setHidden.mutate({ walletId, categoryId: c.id, hidden: !hidden })
                          }
                        >
                          {hidden ? <EyeOff strokeWidth={1.25} /> : <Eye strokeWidth={1.25} />}
                        </Button>
                      </li>
                    );
                  })}
              </ul>
            </div>
          ))}
        </div>
      </Panel>

      <Panel serial="CFG-03 · DANGER" recess className="max-w-2xl">
        <div className="flex flex-col gap-4">
          <p className="text-body-sm text-ink">
            {hasTransactions
              ? "Archiving keeps this wallet's history but removes it everywhere else."
              : "This wallet has no transactions, so it can be deleted."}
          </p>
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button variant="destructive" className="self-start">
                {hasTransactions ? "Archive wallet" : "Delete wallet"}
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>
                  {hasTransactions ? "Archive" : "Delete"} {wallet.name}?
                </AlertDialogTitle>
                <AlertDialogDescription>
                  {hasTransactions
                    ? "You can't add transactions to an archived wallet."
                    : "This can't be undone."}
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction
                  onClick={async () => {
                    if (hasTransactions) await archive.mutateAsync({ id: walletId });
                    else await remove.mutateAsync({ id: walletId });
                    router.push("/wallets");
                  }}
                >
                  {hasTransactions ? "Archive" : "Delete"}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      </Panel>
    </div>
  );
}
