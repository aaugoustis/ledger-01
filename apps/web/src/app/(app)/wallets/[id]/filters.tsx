"use client";

import { type Category, CategoryKind, type Member } from "@ledger-01/api";
import { Button } from "@ledger-01/ui/components/button";
import { CategoryIcon } from "@ledger-01/ui/components/category-icon";
import { Checkbox } from "@ledger-01/ui/components/checkbox";
import { Label } from "@ledger-01/ui/components/label";
import { Popover, PopoverContent, PopoverTrigger } from "@ledger-01/ui/components/popover";
import { parseAmount } from "@ledger-01/ui/lib/money";
import { SlidersHorizontal } from "lucide-react";
import { AmountRange } from "@/components/amount-range";
import { MemberAvatar } from "@/components/member-avatar";
import { NoteSearch } from "@/components/note-search";
import { type Period, today } from "@/lib/period";

export type Filters = {
  categoryIds: string[];
  memberIds: string[];
  note: string;
  min: string;
  max: string;
};

export const noFilters: Filters = { categoryIds: [], memberIds: [], note: "", min: "", max: "" };

/** Builds the API filter from UI state; unparseable amounts are ignored. */
export function filterInput(walletId: string, period: Period, f: Filters, minorUnits: number) {
  const min = f.min ? parseAmount(f.min, minorUnits) : null;
  const max = f.max ? parseAmount(f.max, minorUnits) : null;
  return {
    walletId,
    period,
    categoryIds: f.categoryIds,
    memberIds: f.memberIds,
    noteQuery: f.note.trim(),
    minAmountMinor: min ?? undefined,
    maxAmountMinor: max ?? undefined,
    // Rows after today are upcoming: they show in the Future view, not here.
    today: today(),
  };
}

export function FiltersBar({
  value,
  onChange,
  categories,
  people,
  currency,
}: {
  value: Filters;
  onChange: (f: Filters) => void;
  categories: Category[];
  /** Who can have entered rows; the people filter shows only when there's more than one. */
  people: Member[];
  currency: string;
}) {
  const active =
    value.categoryIds.length + value.memberIds.length + (value.min ? 1 : 0) + (value.max ? 1 : 0);
  const toggle = (id: string) =>
    onChange({
      ...value,
      categoryIds: value.categoryIds.includes(id)
        ? value.categoryIds.filter((c) => c !== id)
        : [...value.categoryIds, id],
    });

  const group = (kind: CategoryKind, title: string) => (
    <fieldset className="flex flex-col gap-1">
      <legend className="hud-sm mb-1 text-ink-soft">{title}</legend>
      {categories
        .filter((c) => c.kind === kind && !c.archived)
        .map((c) => (
          <Label
            key={c.id}
            className="flex cursor-pointer items-center gap-2 rounded-md px-1 py-1 hover:bg-shell-recess"
          >
            <Checkbox
              checked={value.categoryIds.includes(c.id)}
              onCheckedChange={() => toggle(c.id)}
            />
            <CategoryIcon icon={c.icon} color={c.color} size="sm" />
            <span className="truncate text-body-sm">{c.name}</span>
          </Label>
        ))}
    </fieldset>
  );

  return (
    <div className="flex items-center gap-2">
      <NoteSearch value={value.note} onChange={(note) => onChange({ ...value, note })} />
      <Popover>
        <PopoverTrigger asChild>
          <Button variant="secondary" size="sm">
            <SlidersHorizontal strokeWidth={1.25} /> Filters{active ? ` · ${active}` : ""}
          </Button>
        </PopoverTrigger>
        <PopoverContent
          align="end"
          className="flex max-h-[70dvh] w-80 flex-col gap-4 overflow-y-auto"
        >
          <AmountRange
            currency={currency}
            min={value.min}
            max={value.max}
            onChange={(r) => onChange({ ...value, ...r })}
          />
          {people.length > 1 && (
            <PeopleFilter
              people={people}
              value={value.memberIds}
              onChange={(memberIds) => onChange({ ...value, memberIds })}
            />
          )}
          {group(CategoryKind.EXPENSE, "Expense categories")}
          {group(CategoryKind.INCOME, "Income categories")}
          <Button
            variant="ghost"
            size="sm"
            className="self-start"
            onClick={() => onChange(noFilters)}
          >
            Reset filters
          </Button>
        </PopoverContent>
      </Popover>
    </div>
  );
}

/** Checkboxes for who entered the rows. */
export function PeopleFilter({
  people,
  value,
  onChange,
}: {
  people: Member[];
  value: string[];
  onChange: (ids: string[]) => void;
}) {
  return (
    <fieldset className="flex flex-col gap-1">
      <legend className="hud-sm mb-1 text-ink-soft">Entered by</legend>
      {people.map((m) => (
        <Label
          key={m.userId}
          className="flex cursor-pointer items-center gap-2 rounded-md px-1 py-1 hover:bg-shell-recess"
        >
          <Checkbox
            checked={value.includes(m.userId)}
            onCheckedChange={() =>
              onChange(
                value.includes(m.userId)
                  ? value.filter((id) => id !== m.userId)
                  : [...value, m.userId],
              )
            }
          />
          <MemberAvatar member={m} className="size-6 text-body-sm" />
          <span className="truncate text-body-sm">
            {m.name || m.email}
            {m.former ? " (left)" : ""}
          </span>
        </Label>
      ))}
    </fieldset>
  );
}
