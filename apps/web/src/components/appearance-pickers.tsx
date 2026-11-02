"use client";

import { categoryColors, categoryIcons } from "@ledger-01/ui/lib/category-icons";
import { cn } from "@ledger-01/ui/lib/utils";
import { useId } from "react";

// Native radio groups (arrow keys, form semantics) with the inputs visually hidden and the
// labels styled as swatches.

export function IconPicker({
  value,
  onChange,
}: {
  value: string;
  onChange: (icon: string) => void;
}) {
  const name = useId();
  return (
    <fieldset className="grid max-h-48 grid-cols-8 gap-1 overflow-y-auto rounded-md bg-shell-recess p-2 shadow-press">
      <legend className="sr-only">Icon</legend>
      {Object.entries(categoryIcons).map(([icon, Icon]) => (
        <label
          key={icon}
          title={icon}
          className={cn(
            "flex aspect-square cursor-pointer items-center justify-center rounded-sm text-ink hover:bg-shell has-focus-visible:ring-2 has-focus-visible:ring-ring",
            value === icon && "bg-shell shadow-mold ring-2 ring-signal",
          )}
        >
          <input
            type="radio"
            name={name}
            value={icon}
            checked={value === icon}
            onChange={() => onChange(icon)}
            className="sr-only"
          />
          <Icon className="size-4" strokeWidth={1.25} aria-hidden />
          <span className="sr-only">{icon}</span>
        </label>
      ))}
    </fieldset>
  );
}

export function ColorPicker({
  value,
  onChange,
}: {
  value: string;
  onChange: (color: string) => void;
}) {
  const name = useId();
  return (
    <fieldset className="flex flex-wrap gap-2">
      <legend className="sr-only">Color</legend>
      {categoryColors.map((c) => (
        <label
          key={c}
          className={cn(
            "size-8 cursor-pointer rounded-full has-focus-visible:ring-[3px] has-focus-visible:ring-ring/60",
            value === c && "ring-2 ring-ink ring-offset-2 ring-offset-shell",
          )}
          style={{ background: `var(--color-${c})` }}
        >
          <input
            type="radio"
            name={name}
            value={c}
            checked={value === c}
            onChange={() => onChange(c)}
            className="sr-only"
          />
          <span className="sr-only">{c.replace("cat-", "")}</span>
        </label>
      ))}
    </fieldset>
  );
}
