"use client";

import { Button } from "@ledger-01/ui/components/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@ledger-01/ui/components/command";
import { Popover, PopoverContent, PopoverTrigger } from "@ledger-01/ui/components/popover";
import { Check, ChevronsUpDown } from "lucide-react";
import { useState } from "react";

export type MultiOption = { key: string; label: string; icon?: React.ReactNode; group?: string };

/** Filter picker: none selected means "all". */
export function MultiPicker({
  options,
  value,
  onChange,
  allLabel,
  noun,
  searchPlaceholder = "Search…",
}: {
  options: MultiOption[];
  value: string[];
  onChange: (keys: string[]) => void;
  allLabel: string;
  /** Plural noun for the count label, e.g. "wallets". */
  noun: string;
  searchPlaceholder?: string;
}) {
  const [open, setOpen] = useState(false);
  const toggle = (k: string) =>
    onChange(value.includes(k) ? value.filter((v) => v !== k) : [...value, k]);
  const groups = [...new Set(options.map((o) => o.group ?? ""))];
  const label =
    value.length === 0
      ? allLabel
      : value.length === 1
        ? (options.find((o) => o.key === value[0])?.label ?? `1 ${noun}`)
        : `${value.length} ${noun}`;

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="secondary"
          size="sm"
          role="combobox"
          aria-expanded={open}
          className="max-w-56"
        >
          <span className="truncate">{label}</span>
          <ChevronsUpDown className="size-4 text-ink-soft" strokeWidth={1.25} />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-72 p-0">
        <Command>
          <CommandInput placeholder={searchPlaceholder} />
          <CommandList>
            <CommandEmpty>Nothing found.</CommandEmpty>
            {groups.map((g) => (
              <CommandGroup key={g || "all"} heading={g || undefined}>
                {options
                  .filter((o) => (o.group ?? "") === g)
                  .map((o) => (
                    <CommandItem
                      key={o.key}
                      value={`${o.label} ${o.key}`}
                      onSelect={() => toggle(o.key)}
                    >
                      {o.icon}
                      <span className="truncate">{o.label}</span>
                      {value.includes(o.key) && (
                        <Check className="ml-auto size-4" strokeWidth={1.25} />
                      )}
                    </CommandItem>
                  ))}
              </CommandGroup>
            ))}
          </CommandList>
        </Command>
        {value.length > 0 && (
          <div className="border-t border-seam p-2">
            <Button variant="ghost" size="xs" onClick={() => onChange([])}>
              Show {allLabel.toLowerCase()}
            </Button>
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
}
