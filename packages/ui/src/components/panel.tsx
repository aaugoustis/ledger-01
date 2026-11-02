import { cn } from "@ledger-01/ui/lib/utils";
import type * as React from "react";
import { Crosshair } from "./hud";

type Notch = "sm" | "md" | "lg" | false;

const notchClass: Record<Exclude<Notch, false>, string> = {
  sm: "notch-sm",
  md: "notch-md",
  lg: "notch-lg",
};

export type PanelProps = React.ComponentProps<"div"> & {
  /** Microprint in the top-left corner, e.g. "W-01 · USD". */
  serial?: React.ReactNode;
  /** The one chamfered corner (top-right). */
  notch?: Notch;
  /** Sunken shell-recess fill instead of shell. */
  recess?: boolean;
  /** Registration crosshair in the bottom-right corner. */
  cross?: boolean;
  /** Left offset of a signal hinge tab on the top edge, e.g. "24px" or "60%". */
  hinge?: string;
  /** Classes for the inner body (padding, layout). */
  bodyClassName?: string;
};

/**
 * The molded white part: rounded shell, one notched corner, optional serial, crosshair and hinge.
 * The float shadow is a drop-shadow filter on the wrapper so the notch keeps it.
 */
export function Panel({
  serial,
  notch = "md",
  recess,
  cross = false,
  hinge,
  className,
  bodyClassName,
  children,
  ...props
}: PanelProps) {
  return (
    <div data-slot="panel" className={cn("relative drop-float", className)} {...props}>
      {hinge !== undefined && <HingeTab style={{ left: hinge }} />}
      <div
        className={cn(
          "relative h-full rounded-lg p-6 sm:p-8",
          recess ? "bg-shell-recess" : "bg-shell",
          notch && notchClass[notch],
          serial !== undefined && "pt-11",
          bodyClassName,
        )}
      >
        {serial !== undefined && (
          <span className="hud-sm absolute top-3.5 left-6 text-hud-mark">{serial}</span>
        )}
        {children}
        {cross && <Crosshair className="absolute right-3.5 bottom-3.5" />}
      </div>
    </div>
  );
}

/** Small signal stadium sitting on a part's top edge, like a hinge. One per part. */
export function HingeTab({ className, ...props }: React.ComponentProps<"span">) {
  return (
    <span
      aria-hidden
      className={cn("absolute -top-[5px] z-10 h-2.5 w-14 rounded-full bg-signal", className)}
      {...props}
    />
  );
}

/** Frosted clear plastic over color: nav bars and floating toolbars. */
export function GlassPanel({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn("glass rounded-lg p-6", className)} {...props} />;
}
