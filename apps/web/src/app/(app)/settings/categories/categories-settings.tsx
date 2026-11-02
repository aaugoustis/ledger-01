"use client";

import { useMutation } from "@connectrpc/connect-query";
import { type Category, CategoryKind, CategoryService } from "@ledger-01/api";
import { Button } from "@ledger-01/ui/components/button";
import { CategoryIcon } from "@ledger-01/ui/components/category-icon";
import { Checkbox } from "@ledger-01/ui/components/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@ledger-01/ui/components/dialog";
import { Field } from "@ledger-01/ui/components/field";
import { HudLabel } from "@ledger-01/ui/components/hud";
import { Input } from "@ledger-01/ui/components/input";
import { Panel } from "@ledger-01/ui/components/panel";
import { Tabs, TabsList, TabsTrigger } from "@ledger-01/ui/components/tabs";
import { Archive, ArchiveRestore, Merge, Pencil, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { ColorPicker, IconPicker } from "@/components/appearance-pickers";
import { CategoryPicker } from "@/components/category-picker";
import { useCategories } from "@/lib/api/hooks";

type Editing = { mode: "create"; kind: CategoryKind } | { mode: "edit"; category: Category } | null;

export function CategoriesSettings() {
  const { categories } = useCategories();
  const [editing, setEditing] = useState<Editing>(null);
  const [deleting, setDeleting] = useState<Category | null>(null);
  const [merging, setMerging] = useState<string[]>([]);
  const [mergeOpen, setMergeOpen] = useState(false);
  const update = useMutation(CategoryService.method.updateCategory);

  const group = (kind: CategoryKind, title: string) => {
    const list = categories.filter((c) => c.kind === kind);
    return (
      <Panel serial={kind === CategoryKind.EXPENSE ? "CAT-01 · EXPENSE" : "CAT-02 · INCOME"}>
        <div className="flex flex-col gap-3">
          <div className="flex items-center justify-between gap-2">
            <HudLabel rule={false}>{title}</HudLabel>
            <Button
              size="sm"
              variant="secondary"
              onClick={() => setEditing({ mode: "create", kind })}
            >
              <Plus strokeWidth={1.25} /> Add
            </Button>
          </div>
          <ul className="flex flex-col">
            {list.map((c) => (
              <li
                key={c.id}
                className="flex items-center gap-3 border-b border-seam/60 py-2 last:border-b-0"
              >
                <Checkbox
                  aria-label={`Select ${c.name} to merge`}
                  checked={merging.includes(c.id)}
                  onCheckedChange={() =>
                    setMerging((m) =>
                      m.includes(c.id) ? m.filter((x) => x !== c.id) : [...m, c.id],
                    )
                  }
                />
                <CategoryIcon icon={c.icon} color={c.color} size="sm" />
                <span
                  className={
                    c.archived
                      ? "flex-1 text-body-sm text-ink-soft"
                      : "flex-1 text-body-sm text-ink"
                  }
                >
                  {c.name}
                  {c.archived && <span className="hud-sm ml-2 text-hud-mark">archived</span>}
                </span>
                <span className="hud-sm hidden text-ink-soft sm:inline">
                  {c.transactionCount.toString()} txn · {c.walletCount.toString()} wallets
                </span>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label={`Edit ${c.name}`}
                  onClick={() => setEditing({ mode: "edit", category: c })}
                >
                  <Pencil strokeWidth={1.25} />
                </Button>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label={c.archived ? `Restore ${c.name}` : `Archive ${c.name}`}
                  onClick={() => update.mutate({ id: c.id, archived: !c.archived })}
                >
                  {c.archived ? (
                    <ArchiveRestore strokeWidth={1.25} />
                  ) : (
                    <Archive strokeWidth={1.25} />
                  )}
                </Button>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label={`Delete ${c.name}`}
                  onClick={() => setDeleting(c)}
                >
                  <Trash2 strokeWidth={1.25} />
                </Button>
              </li>
            ))}
          </ul>
        </div>
      </Panel>
    );
  };

  return (
    <div className="flex flex-col gap-6">
      {merging.length > 0 && (
        <div className="glass sticky top-24 z-30 flex items-center justify-between rounded-full py-2 pr-2 pl-5">
          <span className="text-label text-ink">{merging.length} selected</span>
          <div className="flex gap-2">
            <Button variant="ghost" size="sm" onClick={() => setMerging([])}>
              Clear
            </Button>
            <Button size="sm" onClick={() => setMergeOpen(true)}>
              <Merge strokeWidth={1.25} /> Merge into…
            </Button>
          </div>
        </div>
      )}
      {group(CategoryKind.EXPENSE, "Expense categories")}
      {group(CategoryKind.INCOME, "Income categories")}

      <CategoryDialog editing={editing} onClose={() => setEditing(null)} />
      <DeleteCategoryDialog
        category={deleting}
        categories={categories}
        onClose={() => setDeleting(null)}
      />
      <MergeDialog
        open={mergeOpen}
        sources={merging}
        categories={categories}
        onClose={(done) => {
          setMergeOpen(false);
          if (done) setMerging([]);
        }}
      />
    </div>
  );
}

function CategoryDialog({ editing, onClose }: { editing: Editing; onClose: () => void }) {
  const create = useMutation(CategoryService.method.createCategory);
  const update = useMutation(CategoryService.method.updateCategory);
  const initial = editing?.mode === "edit" ? editing.category : undefined;
  const [kind, setKind] = useState<CategoryKind>(CategoryKind.EXPENSE);
  const [name, setName] = useState("");
  const [icon, setIcon] = useState("ellipsis");
  const [color, setColor] = useState("cat-slate");
  const [key, setKey] = useState<Editing>(null);

  // Reset the form whenever a different category (or "create") is opened.
  if (editing !== key) {
    setKey(editing);
    setKind(editing?.mode === "create" ? editing.kind : (initial?.kind ?? CategoryKind.EXPENSE));
    setName(initial?.name ?? "");
    setIcon(initial?.icon ?? "ellipsis");
    setColor(initial?.color ?? "cat-slate");
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (initial) await update.mutateAsync({ id: initial.id, name, icon, color });
    else await create.mutateAsync({ kind, name, icon, color });
    toast.success(initial ? "Category saved" : "Category added");
    onClose();
  }

  return (
    <Dialog open={editing !== null} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <form onSubmit={submit} className="flex flex-col gap-5">
          <DialogHeader>
            <DialogTitle>{initial ? "Edit category" : "New category"}</DialogTitle>
            <DialogDescription>Categories are shared by every wallet.</DialogDescription>
          </DialogHeader>
          <div className="flex items-center gap-4">
            <CategoryIcon icon={icon} color={color} size="lg" />
            <Field label="Name" htmlFor="cat-name" className="flex-1">
              <Input
                id="cat-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                maxLength={40}
                required
              />
            </Field>
          </div>
          {!initial && (
            <Tabs
              value={kind === CategoryKind.INCOME ? "income" : "expense"}
              onValueChange={(v) =>
                setKind(v === "income" ? CategoryKind.INCOME : CategoryKind.EXPENSE)
              }
            >
              <TabsList className="w-full">
                <TabsTrigger value="expense">Expense</TabsTrigger>
                <TabsTrigger value="income">Income</TabsTrigger>
              </TabsList>
            </Tabs>
          )}
          <Field label="Icon">
            <IconPicker value={icon} onChange={setIcon} />
          </Field>
          <Field label="Color">
            <ColorPicker value={color} onChange={setColor} />
          </Field>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={!name.trim() || create.isPending || update.isPending}>
              {initial ? "Save" : "Add category"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function DeleteCategoryDialog({
  category,
  categories,
  onClose,
}: {
  category: Category | null;
  categories: Category[];
  onClose: () => void;
}) {
  const del = useMutation(CategoryService.method.deleteCategory);
  const [replacement, setReplacement] = useState("");
  const used = (category?.transactionCount ?? 0n) > 0n;
  const options = categories.filter((c) => c.kind === category?.kind && c.id !== category?.id);

  return (
    <Dialog
      open={category !== null}
      onOpenChange={(o) => {
        if (o) return;
        setReplacement("");
        onClose();
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete {category?.name}?</DialogTitle>
          <DialogDescription>
            {used
              ? `${category?.transactionCount} transactions use it. Move them to another category first, or archive it instead.`
              : "No transactions use it."}
          </DialogDescription>
        </DialogHeader>
        {used && (
          <Field label="Move transactions to">
            <CategoryPicker
              categories={options}
              walletId=""
              value={replacement}
              onChange={setReplacement}
            />
          </Field>
        )}
        <DialogFooter>
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            disabled={(used && !replacement) || del.isPending}
            onClick={async () => {
              if (!category) return;
              await del.mutateAsync({
                id: category.id,
                replacementId: used ? replacement : undefined,
              });
              setReplacement("");
              onClose();
            }}
          >
            Delete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function MergeDialog({
  open,
  sources,
  categories,
  onClose,
}: {
  open: boolean;
  sources: string[];
  categories: Category[];
  onClose: (done: boolean) => void;
}) {
  const merge = useMutation(CategoryService.method.mergeCategories);
  const [target, setTarget] = useState("");
  const selected = categories.filter((c) => sources.includes(c.id));
  const kinds = new Set(selected.map((c) => c.kind));
  const kind = selected[0]?.kind;
  const options = categories.filter((c) => c.kind === kind && !sources.includes(c.id));

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose(false)}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Merge {selected.length} categories</DialogTitle>
          <DialogDescription>
            {kinds.size > 1
              ? "Pick categories of one kind (all expense or all income)."
              : `Their transactions move to the target and ${selected.map((c) => c.name).join(", ")} are deleted.`}
          </DialogDescription>
        </DialogHeader>
        {kinds.size === 1 && (
          <Field label="Merge into">
            <CategoryPicker categories={options} walletId="" value={target} onChange={setTarget} />
          </Field>
        )}
        <DialogFooter>
          <Button variant="ghost" onClick={() => onClose(false)}>
            Cancel
          </Button>
          <Button
            disabled={kinds.size !== 1 || !target || merge.isPending}
            onClick={async () => {
              const res = await merge.mutateAsync({ sourceIds: sources, targetId: target });
              toast.success(`Moved ${res.movedTransactionCount} transactions`);
              setTarget("");
              onClose(true);
            }}
          >
            Merge
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
