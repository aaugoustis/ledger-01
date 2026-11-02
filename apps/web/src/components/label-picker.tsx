"use client";

import { useMutation } from "@connectrpc/connect-query";
import { type Label, LabelService } from "@ledger-01/api";
import { Button } from "@ledger-01/ui/components/button";
import { ColorDot } from "@ledger-01/ui/components/category-icon";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@ledger-01/ui/components/command";
import { Popover, PopoverContent, PopoverTrigger } from "@ledger-01/ui/components/popover";
import { categoryColors } from "@ledger-01/ui/lib/category-icons";
import { Check, ChevronsUpDown, Plus } from "lucide-react";
import { useState } from "react";

/** Multi-select for labels with inline "create label". */
export function LabelPicker({
  labels,
  value,
  onChange,
  id,
}: {
  labels: Label[];
  value: string[];
  onChange: (ids: string[]) => void;
  id?: string;
}) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const create = useMutation(LabelService.method.createLabel);
  const selected = labels.filter((l) => value.includes(l.id));
  const exact = labels.some((l) => l.name.toLowerCase() === query.trim().toLowerCase());

  const toggle = (labelId: string) =>
    onChange(value.includes(labelId) ? value.filter((v) => v !== labelId) : [...value, labelId]);

  async function createLabel() {
    const name = query.trim();
    if (!name) return;
    const color = categoryColors[labels.length % categoryColors.length] ?? "cat-slate";
    const res = await create.mutateAsync({ name, color });
    if (res.label) onChange([...value, res.label.id]);
    setQuery("");
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          id={id}
          type="button"
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full justify-between px-3"
        >
          {selected.length ? (
            <span className="flex min-w-0 items-center gap-1.5 overflow-hidden">
              {selected.map((l) => (
                <span key={l.id} className="flex shrink-0 items-center gap-1 text-body-sm">
                  <ColorDot color={l.color} />
                  {l.name}
                </span>
              ))}
            </span>
          ) : (
            <span className="text-ink-soft">No labels</span>
          )}
          <ChevronsUpDown className="size-4 shrink-0 text-ink-soft" strokeWidth={1.25} />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-(--radix-popover-trigger-width) min-w-64 p-0" align="start">
        <Command>
          <CommandInput placeholder="Search or create…" value={query} onValueChange={setQuery} />
          <CommandList>
            <CommandEmpty>{query.trim() ? "No label yet." : "No labels yet."}</CommandEmpty>
            <CommandGroup>
              {labels.map((l) => (
                <CommandItem key={l.id} value={l.name} onSelect={() => toggle(l.id)}>
                  <ColorDot color={l.color} />
                  <span className="truncate">{l.name}</span>
                  {value.includes(l.id) && <Check className="ml-auto size-4" strokeWidth={1.25} />}
                </CommandItem>
              ))}
            </CommandGroup>
            {query.trim() && !exact && (
              <CommandGroup>
                <CommandItem
                  value={`create ${query}`}
                  onSelect={createLabel}
                  disabled={create.isPending}
                >
                  <Plus className="size-4" strokeWidth={1.25} />
                  Add label “{query.trim()}”
                </CommandItem>
              </CommandGroup>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
