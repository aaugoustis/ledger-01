"use client";

import { Input } from "@ledger-01/ui/components/input";
import { Search } from "lucide-react";
import { useEffect, useRef, useState } from "react";

/**
 * Note search box that reports after typing pauses, so typing doesn't fire a request per
 * keystroke. Its text stays at 16px so phones don't zoom in on focus.
 */
export function NoteSearch({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const [note, setNote] = useState(value);
  // Callers pass a new callback each render; only typing should restart the wait.
  const report = useRef(onChange);
  report.current = onChange;
  // A reset from outside (filters cleared) empties the box.
  useEffect(() => {
    if (value === "") setNote("");
  }, [value]);
  useEffect(() => {
    const t = setTimeout(() => {
      if (note !== value) report.current(note);
    }, 300);
    return () => clearTimeout(t);
  }, [note, value]);

  return (
    <div className="relative">
      <Search
        className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-ink-soft"
        strokeWidth={1.25}
      />
      <Input
        aria-label="Search notes"
        placeholder="Search notes"
        value={note}
        onChange={(e) => setNote(e.target.value)}
        className="h-9 w-44 pl-9 sm:w-56"
      />
    </div>
  );
}
