import { cn } from "@ledger-01/ui/lib/utils";
import type * as React from "react";
import { Label } from "./label";

/** The fault LED: a small danger dot with a tint halo that blinks as it appears. */
export function FaultLed({ className }: { className?: string }) {
  return (
    <span
      aria-hidden
      className={cn(
        "led-blink inline-block size-1.5 shrink-0 rounded-full bg-danger shadow-[0_0_0_3px_var(--color-danger-tint)]",
        className,
      )}
    />
  );
}

/**
 * Announces an error to screen readers without showing a line under the control; the visible
 * cue is the red label with its fault LED. Renders nothing without children.
 */
export function FieldError({ children }: { children?: React.ReactNode }) {
  if (!children) return null;
  return (
    <span role="alert" className="sr-only">
      {children}
    </span>
  );
}

/** Field label; when invalid it turns danger with a fault LED in front. */
export function FieldLabel({
  htmlFor,
  invalid,
  className,
  children,
}: {
  htmlFor?: string;
  invalid?: boolean;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <Label
      htmlFor={htmlFor}
      data-invalid={invalid || undefined}
      className={cn("text-body-sm", invalid ? "text-danger" : "text-ink-soft", className)}
    >
      {invalid && <FaultLed />}
      {children}
    </Label>
  );
}

/** Label + control + help text, stacked. Pass the control's id as `htmlFor`. */
export function Field({
  label,
  htmlFor,
  error,
  help,
  className,
  children,
}: {
  label: React.ReactNode;
  htmlFor?: string;
  /** Turns the label red with a fault LED; the message is announced, not shown. */
  error?: React.ReactNode;
  help?: React.ReactNode;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div className={cn("flex min-w-0 flex-col gap-1.5", className)}>
      <FieldLabel htmlFor={htmlFor} invalid={Boolean(error)}>
        {label}
      </FieldLabel>
      {children}
      {error ? (
        <FieldError>{error}</FieldError>
      ) : help ? (
        <p className="text-body-sm text-ink-soft">{help}</p>
      ) : null}
    </div>
  );
}
