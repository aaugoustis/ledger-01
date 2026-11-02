import { cn } from "@ledger-01/ui/lib/utils";
import type * as React from "react";

// Inputs are sunken wells: shell-recess fill, pressed shadow, radius-md.
function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        "h-control-height w-full min-w-0 rounded-md border border-transparent bg-shell-recess px-4 text-body-md text-ink shadow-press transition-[box-shadow,border-color] outline-none placeholder:text-ink-soft disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50",
        "file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-label file:text-ink",
        "focus-visible:border-signal-ink focus-visible:ring-[3px] focus-visible:ring-ring/30",
        "aria-invalid:border-danger aria-invalid:bg-danger-tint aria-invalid:ring-danger/20",
        className,
      )}
      {...props}
    />
  );
}

export { Input };
