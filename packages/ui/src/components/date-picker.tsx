"use client";

import { focusNextField } from "@ledger-01/ui/lib/form-flow";
import { cn } from "@ledger-01/ui/lib/utils";
import { addDays, format, isSameDay } from "date-fns";
import { CalendarDays } from "lucide-react";
import { useRef, useState } from "react";
import { Button } from "./button";
import { Calendar } from "./calendar";
import { Popover, PopoverContent, PopoverTrigger } from "./popover";

/** "YYYY-MM-DD" -> local Date (no time-zone shift). */
export function parseISODate(value: string): Date | undefined {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (!m) return undefined;
  return new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]));
}

/** Local Date -> "YYYY-MM-DD". */
export function toISODate(date: Date): string {
  return format(date, "yyyy-MM-dd");
}

/** Short field label: "Today, Oct 29", "Yesterday, Oct 28", "Tue, Oct 20", "Mon, Dec 1, 2025". */
export function dateLabel(date: Date, today: Date): string {
  if (isSameDay(date, today)) return `Today, ${format(date, "MMM d")}`;
  if (isSameDay(date, addDays(today, -1))) return `Yesterday, ${format(date, "MMM d")}`;
  return format(
    date,
    date.getFullYear() === today.getFullYear() ? "EEE, MMM d" : "EEE, MMM d, yyyy",
  );
}

/**
 * A calendar date in a sunken well, opening a Module/01 Calendar with Today and Yesterday
 * shortcuts. Value and onChange use "YYYY-MM-DD" strings.
 */
export function DatePicker({
  id,
  value,
  onChange,
  placeholder = "Pick a date",
  disabled,
  className,
}: {
  id?: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
}) {
  const [open, setOpen] = useState(false);
  const selected = parseISODate(value);
  const [month, setMonth] = useState<Date>(selected ?? new Date());
  const today = new Date();
  const trigger = useRef<HTMLButtonElement>(null);
  const committed = useRef(false);

  function openChange(next: boolean) {
    if (next) setMonth(selected ?? new Date());
    setOpen(next);
  }

  function pick(date: Date) {
    onChange(toISODate(date));
    committed.current = true;
    setOpen(false);
  }

  const label = selected ? dateLabel(selected, today) : placeholder;

  return (
    <Popover open={open} onOpenChange={openChange}>
      <PopoverTrigger asChild>
        <button
          ref={trigger}
          data-flow-field
          id={id}
          type="button"
          disabled={disabled}
          className={cn(
            "flex h-control-height w-full min-w-0 cursor-pointer items-center gap-3 rounded-md border border-transparent bg-shell-recess px-4 text-left text-body-md text-ink shadow-press outline-none transition-[box-shadow,border-color] disabled:pointer-events-none disabled:opacity-50",
            "focus-visible:border-signal-ink focus-visible:ring-[3px] focus-visible:ring-ring/30 data-[state=open]:border-signal-ink",
            className,
          )}
        >
          <CalendarDays className="size-4 shrink-0 text-ink-soft" strokeWidth={1.25} />
          <span className={cn("truncate", !selected && "text-ink-soft")}>{label}</span>
        </button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        className="w-auto rounded-lg p-3"
        onCloseAutoFocus={(e) => {
          // Form flow: after picking a date, move on to the next field.
          if (!committed.current) return;
          committed.current = false;
          e.preventDefault();
          focusNextField(trigger.current);
        }}
      >
        <Calendar
          mode="single"
          required
          selected={selected}
          onSelect={pick}
          month={month}
          onMonthChange={setMonth}
          autoFocus
        />
        <div className="mt-3 flex gap-2 border-t border-seam pt-3">
          <Button type="button" size="xs" variant="secondary" onClick={() => pick(today)}>
            Today
          </Button>
          <Button
            type="button"
            size="xs"
            variant="secondary"
            onClick={() => pick(addDays(today, -1))}
          >
            Yesterday
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}
