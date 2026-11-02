"use client";

import { focusNextField } from "@ledger-01/ui/lib/form-flow";
import { cn } from "@ledger-01/ui/lib/utils";
import { Check, ChevronsUpDown } from "lucide-react";
import { useRef, useState } from "react";
import { Button } from "./button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "./command";
import { Popover, PopoverContent, PopoverTrigger } from "./popover";

export type ComboboxGroup<T> = { heading?: string; options: T[] };

/**
 * Searchable single-choice picker. Part of the form flow: typing on the focused trigger opens it
 * with that text as the search, and picking an option moves focus to the next field.
 */
export function Combobox<T>({
  id,
  value,
  onChange,
  options,
  groups,
  getKey,
  getSearchText,
  renderOption,
  renderValue,
  placeholder = "Choose…",
  searchPlaceholder = "Search…",
  emptyText = "Nothing found.",
  footer,
  invalid,
  disabled,
  className,
  contentClassName,
}: {
  id?: string;
  value: string;
  onChange: (key: string) => void;
  /** Flat options; or pass `groups` for headed sections. */
  options?: T[];
  groups?: ComboboxGroup<T>[];
  getKey: (option: T) => string;
  /** Text the search matches against (defaults to the key). */
  getSearchText?: (option: T) => string;
  renderOption: (option: T) => React.ReactNode;
  /** The selected option on the trigger (defaults to renderOption). */
  renderValue?: (option: T) => React.ReactNode;
  placeholder?: React.ReactNode;
  searchPlaceholder?: string;
  emptyText?: React.ReactNode;
  /** Shown under the list, e.g. a link to manage the options. */
  footer?: React.ReactNode;
  invalid?: boolean;
  disabled?: boolean;
  className?: string;
  contentClassName?: string;
}) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const trigger = useRef<HTMLButtonElement>(null);
  const committed = useRef(false);

  const sections: ComboboxGroup<T>[] = groups ?? [{ options: options ?? [] }];
  const current = sections.flatMap((g) => g.options).find((o) => getKey(o) === value);
  const search = getSearchText ?? getKey;

  function openWith(text: string) {
    setQuery(text);
    setOpen(true);
  }

  function pick(option: T) {
    committed.current = true;
    onChange(getKey(option));
    setOpen(false);
  }

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
          ref={trigger}
          id={id}
          type="button"
          variant="outline"
          role="combobox"
          data-flow-autoopen={value ? undefined : ""}
          aria-expanded={open}
          aria-invalid={invalid || undefined}
          disabled={disabled}
          className={cn("w-full justify-between px-3", className)}
          onKeyDown={(e) => {
            // Type to search: a printable key on the closed trigger opens it with that letter.
            if (
              !open &&
              e.key.length === 1 &&
              !e.ctrlKey &&
              !e.metaKey &&
              !e.altKey &&
              e.key !== " "
            ) {
              e.preventDefault();
              openWith(e.key);
            }
          }}
        >
          {current ? (
            <span className="flex min-w-0 items-center gap-2">
              {(renderValue ?? renderOption)(current)}
            </span>
          ) : (
            <span className="text-ink-soft">{placeholder}</span>
          )}
          <ChevronsUpDown className="size-4 shrink-0 text-ink-soft" strokeWidth={1.25} />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        className={cn("w-(--radix-popover-trigger-width) min-w-64 p-0", contentClassName)}
        onCloseAutoFocus={(e) => {
          // Form flow: after a pick, move on to the next field (Escape returns to the trigger).
          if (!committed.current) return;
          committed.current = false;
          e.preventDefault();
          focusNextField(trigger.current);
        }}
      >
        <Command>
          <CommandInput placeholder={searchPlaceholder} value={query} onValueChange={setQuery} />
          <CommandList>
            <CommandEmpty>{emptyText}</CommandEmpty>
            {sections.map((g, i) => (
              <CommandGroup key={g.heading ?? i} heading={g.heading}>
                {g.options.map((o) => (
                  <CommandItem
                    key={getKey(o)}
                    value={`${search(o)} ${getKey(o)}`}
                    onSelect={() => pick(o)}
                  >
                    {renderOption(o)}
                    {getKey(o) === value && <Check className="ml-auto size-4" strokeWidth={1.25} />}
                  </CommandItem>
                ))}
              </CommandGroup>
            ))}
          </CommandList>
        </Command>
        {footer && <div className="border-t border-seam p-2">{footer}</div>}
      </PopoverContent>
    </Popover>
  );
}
