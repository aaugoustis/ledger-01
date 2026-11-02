"use client";

import {
  CircleCheckIcon,
  InfoIcon,
  Loader2Icon,
  OctagonXIcon,
  TriangleAlertIcon,
} from "lucide-react";
import type * as React from "react";
import { Toaster as Sonner, type ToasterProps } from "sonner";

// Light theme only (Module/01 is lit like a white studio).
const Toaster = ({ ...props }: ToasterProps) => {
  return (
    <Sonner
      theme="light"
      className="toaster group"
      icons={{
        success: <CircleCheckIcon className="size-4 text-income-ink" strokeWidth={1.25} />,
        info: <InfoIcon className="size-4" strokeWidth={1.25} />,
        warning: <TriangleAlertIcon className="size-4 text-signal-ink" strokeWidth={1.25} />,
        error: <OctagonXIcon className="size-4 text-danger" strokeWidth={1.25} />,
        loading: <Loader2Icon className="size-4 animate-spin" strokeWidth={1.25} />,
      }}
      toastOptions={{ classNames: { toast: "font-sans text-body-sm shadow-float" } }}
      style={
        {
          "--normal-bg": "var(--color-shell)",
          "--normal-text": "var(--color-ink)",
          "--normal-border": "var(--color-seam)",
          "--border-radius": "var(--radius-lg)",
        } as React.CSSProperties
      }
      {...props}
    />
  );
};

export { Toaster };
