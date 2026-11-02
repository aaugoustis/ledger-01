import { cn } from "@ledger-01/ui/lib/utils";
import type * as React from "react";

/** 12px plus-mark used as a registration mark on corners and frames. Decorative. */
export function Crosshair({
  tone,
  className,
  ...props
}: React.ComponentProps<"svg"> & { tone?: "signal" }) {
  return (
    <svg
      aria-hidden
      viewBox="0 0 12 12"
      className={cn(
        "size-3 shrink-0",
        tone === "signal" ? "text-signal-ink" : "text-hud-mark",
        className,
      )}
      {...props}
    >
      <path d="M6 0v12M0 6h12" stroke="currentColor" strokeWidth="1.25" fill="none" />
      <circle cx="6" cy="6" r="2.5" stroke="currentColor" strokeWidth="1.25" fill="none" />
    </svg>
  );
}

/** Mono overline: signal code chip, label, a seam rule and an optional end marker. */
export function HudLabel({
  code,
  end,
  rule = true,
  className,
  children,
  ...props
}: React.ComponentProps<"div"> & { code?: string; end?: React.ReactNode; rule?: boolean }) {
  return (
    <div className={cn("hud flex items-center gap-3 text-ink-soft", className)} {...props}>
      {code && <span className="rounded-xs bg-signal px-1.5 py-0.5 text-ink">{code}</span>}
      <span>{children}</span>
      {rule && <span aria-hidden className="h-px min-w-6 flex-1 bg-seam" />}
      {end !== undefined && <span>{end}</span>}
    </div>
  );
}

/** Measurement-tape strip: major ticks every 40px, minor every 8px. Decorative. */
export function TickRuler({ labels, className }: { labels?: string[]; className?: string }) {
  return (
    <div aria-hidden className={cn("text-hud-mark", className)}>
      <div
        className="h-3.5"
        style={{
          backgroundImage:
            "repeating-linear-gradient(90deg, var(--color-hud-mark) 0 1px, transparent 1px 40px), repeating-linear-gradient(90deg, var(--color-seam) 0 1px, transparent 1px 8px)",
          backgroundSize: "100% 14px, 100% 7px",
          backgroundRepeat: "no-repeat",
        }}
      />
      {labels && labels.length > 0 && (
        <div className="hud-sm mt-0.5 flex justify-between">
          {labels.map((l) => (
            <span key={l}>{l}</span>
          ))}
        </div>
      )}
    </div>
  );
}

/** LED dot with a word next to it. Color never carries the meaning alone. */
export function StatusLed({
  tone = "on",
  className,
  children,
  ...props
}: React.ComponentProps<"span"> & { tone?: "on" | "signal" | "off" }) {
  return (
    <span className={cn("hud inline-flex items-center gap-2 text-ink", className)} {...props}>
      <span
        aria-hidden
        className={cn(
          "size-2 shrink-0 rounded-full",
          tone === "on" && "bg-led-on shadow-[0_0_0_3px_rgba(31,164,90,0.18)]",
          tone === "signal" && "bg-signal shadow-[0_0_0_3px_var(--color-signal-tint)]",
          tone === "off" && "bg-seam",
        )}
      />
      {children}
    </span>
  );
}

/** Pegboard slots: the product grill. Decorative. */
export function Grill({
  count = 6,
  tone,
  className,
}: {
  count?: number;
  tone?: "signal" | "on-signal";
  className?: string;
}) {
  return (
    <div aria-hidden className={cn("flex flex-wrap gap-2.5", className)}>
      {Array.from({ length: count }, (_, i) => (
        <span
          // biome-ignore lint/suspicious/noArrayIndexKey: static decorative slots
          key={i}
          className={cn(
            "h-2.5 w-9 rounded-full",
            tone === "on-signal"
              ? "bg-signal-shade shadow-[inset_0_2px_4px_rgba(0,0,0,0.18)]"
              : tone === "signal"
                ? "bg-signal shadow-[inset_0_2px_4px_rgba(0,0,0,0.14)]"
                : "bg-shell-recess shadow-press",
          )}
        />
      ))}
    </div>
  );
}

/** Section heading: HUD label with code above a Michroma title and an optional aside. */
export function SectionHeader({
  title,
  code,
  label,
  aside,
  ruler = false,
  as: Tag = "h1",
  className,
}: {
  title: React.ReactNode;
  code?: string;
  label?: React.ReactNode;
  aside?: React.ReactNode;
  ruler?: boolean;
  as?: "h1" | "h2" | "h3";
  className?: string;
}) {
  return (
    <div className={cn("flex flex-col gap-3", className)}>
      {(code || label) && <HudLabel code={code}>{label}</HudLabel>}
      <div className="flex flex-wrap items-end justify-between gap-4">
        <Tag className="font-display text-display-md text-ink sm:text-display-lg">{title}</Tag>
        {aside}
      </div>
      {ruler && <TickRuler />}
    </div>
  );
}
