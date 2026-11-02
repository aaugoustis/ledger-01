"use client";

import { cn } from "@ledger-01/ui/lib/utils";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { type ChevronProps, DayPicker, type DayPickerProps } from "react-day-picker";

function CalendarChevron({ orientation, className }: ChevronProps) {
  const Icon = orientation === "left" ? ChevronLeft : ChevronRight;
  return <Icon className={cn("size-4", className)} strokeWidth={1.25} />;
}

const navButton =
  "inline-flex size-9 cursor-pointer items-center justify-center rounded-full bg-shell text-ink shadow-mold outline-none hover:bg-shell-recess focus-visible:ring-[3px] focus-visible:ring-ring/60 active:shadow-press disabled:pointer-events-none disabled:opacity-40";

/**
 * Module/01 month calendar: HUD caption, mono weekday microprint, stadium days. The selected
 * day is a signal part with ink text; today carries a signal-ink ring.
 */
export function Calendar({
  className,
  classNames,
  showOutsideDays = true,
  weekStartsOn = 1,
  ...props
}: DayPickerProps) {
  return (
    <DayPicker
      showOutsideDays={showOutsideDays}
      weekStartsOn={weekStartsOn}
      className={cn("select-none", className)}
      classNames={{
        root: "relative",
        months: "relative flex flex-col",
        month: "flex flex-col gap-2",
        month_caption: "flex h-9 items-center pl-1",
        caption_label: "hud text-ink",
        nav: "absolute top-0 right-0 z-10 flex items-center gap-1.5",
        button_previous: navButton,
        button_next: navButton,
        month_grid: "border-collapse",
        weekdays: "",
        weekday: "hud-sm h-8 w-10 text-center font-normal text-ink-soft",
        weeks: "",
        week: "",
        day: "p-0.5 text-center",
        day_button:
          "tabular mx-auto flex size-9 cursor-pointer items-center justify-center rounded-full text-body-sm text-ink outline-none transition-[background-color,box-shadow] hover:bg-shell-recess focus-visible:ring-[3px] focus-visible:ring-ring/60",
        selected:
          "[&>button]:bg-signal [&>button]:font-bold [&>button]:text-ink [&>button]:shadow-mold [&>button]:hover:bg-signal",
        today:
          "[&>button]:font-bold [&>button]:text-signal-ink [&>button]:ring-1 [&>button]:ring-signal-ink",
        outside: "[&>button]:text-hud-mark",
        disabled: "[&>button]:pointer-events-none [&>button]:opacity-40",
        hidden: "invisible",
        ...classNames,
      }}
      components={{ Chevron: CalendarChevron }}
      {...props}
    />
  );
}
