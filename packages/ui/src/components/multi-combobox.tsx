"use client";

import { cn } from "@ledger-01/ui/lib/utils";
import { Check, ChevronsUpDown } from "lucide-react";
import { useState } from "react";
import { Button } from "./button";
import type { ComboboxGroup } from "./combobox";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "./command";
import { Popover, PopoverContent, PopoverTrigger } from "./popover";

/**
 * Searchable multi-choice picker where nothing selected means "all". Stays open while you
 * toggle options; Escape or a click outside closes it back onto the trigger, and Tab moves on.
 * Typing on the focused trigger opens it with that text as the search, like Combobox.
 */
export function MultiCombobox<T>({
  id,
  value,
  onChange,
  options,
  groups,
  getKey,
  getSearchText,
  renderOption,
  allLabel,
  noun,
  searchPlaceholder = "Search…",
  emptyText = "Nothing found.",
  invalid,
  disabled,
  className,
  contentClassName,
}: {
  id?: string;
  value: string[];
  onChange: (keys: string[]) => void;
  options?: T[];
  groups?: ComboboxGroup<T>[];
  getKey: (option: T) => string;
  getSearchText?: (option: T) => string;
  renderOption: (option: T) => React.ReactNode;
  /** Trigger text when nothing is selected, e.g. "All wallets". */
  allLabel: string;
  /** Plural noun for the count, e.g. "wallets" → "3 wallets". */
  noun: string;
  searchPlaceholder?: string;
  emptyText?: React.ReactNode;
  invalid?: boolean;
  disabled?: boolean;
  className?: string;
  contentClassName?: string;
}) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const sections: ComboboxGroup<T>[] = groups ?? [{ options: options ?? [] }];
  const all = sections.flatMap((g) => g.options);
  const search = getSearchText ?? getKey;
  const only = value.length === 1 ? all.find((o) => getKey(o) === value[0]) : undefined;
  const toggle = (k: string) =>
    onChange(value.includes(k) ? value.filter((v) => v !== k) : [...value, k]);

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        if (next) setQuery("");
        setOpen(next);
      }}
    >
      <PopoverTrigger asChild>
        <Button
          id={id}
          type="button"
          variant="outline"
          role="combobox"
          aria-expanded={open}
          aria-invalid={invalid || undefined}
          disabled={disabled}
          className={cn("w-full justify-between px-3", className)}
          onKeyDown={(e) => {
            if (
              !open &&
              e.key.length === 1 &&
              !e.ctrlKey &&
              !e.metaKey &&
              !e.altKey &&
              e.key !== " "
            ) {
              e.preventDefault();
              setQuery(e.key);
              setOpen(true);
            }
          }}
        >
          <span className="flex min-w-0 items-center gap-2">
            {value.length === 0 ? (
              <span className="truncate">{allLabel}</span>
            ) : only ? (
              renderOption(only)
            ) : (
              <span className="truncate">
                {value.length} {noun}
              </span>
            )}
          </span>
          <ChevronsUpDown className="size-4 shrink-0 text-ink-soft" strokeWidth={1.25} />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        className={cn("w-(--radix-popover-trigger-width) min-w-64 p-0", contentClassName)}
      >
        <Command>
          <CommandInput placeholder={searchPlaceholder} value={query} onValueChange={setQuery} />
          <CommandList>
            <CommandEmpty>{emptyText}</CommandEmpty>
            {sections.map((g, i) => (
              <CommandGroup key={g.heading ?? i} heading={g.heading}>
                {g.options.map((o) => {
                  const k = getKey(o);
                  return (
                    <CommandItem
                      key={k}
                      value={`${search(o)} ${k}`}
                      onSelect={() => toggle(k)}
                      aria-selected={value.includes(k)}
                    >
                      {renderOption(o)}
                      {value.includes(k) && <Check className="ml-auto size-4" strokeWidth={1.25} />}
                    </CommandItem>
                  );
                })}
              </CommandGroup>
            ))}
          </CommandList>
        </Command>
        {value.length > 0 && (
          <div className="border-t border-seam p-2">
            <Button type="button" variant="ghost" size="xs" onClick={() => onChange([])}>
              Use {allLabel.toLowerCase()}
            </Button>
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
}
