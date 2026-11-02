"use client";

import { cn } from "@ledger-01/ui/lib/utils";
import Link from "next/link";
import { usePathname } from "next/navigation";

const items = [
  { href: "/settings/categories", label: "Categories" },
  { href: "/settings/labels", label: "Labels" },
  { href: "/settings/import-rules", label: "Import rules" },
  { href: "/settings/connected-apps", label: "Connected apps" },
  { href: "/settings/household", label: "Household" },
  { href: "/settings/data", label: "Data" },
  { href: "/settings/account", label: "Account" },
];

export function SettingsNav() {
  const path = usePathname();
  return (
    <nav aria-label="Settings" className="flex gap-1 overflow-x-auto md:flex-col">
      {items.map((item, i) => (
        <Link
          key={item.href}
          href={item.href}
          aria-current={path === item.href ? "page" : undefined}
          className={cn(
            "flex items-center gap-2 rounded-full px-4 py-2 text-label whitespace-nowrap text-ink hover:bg-shell-recess",
            path === item.href && "bg-shell text-signal-ink shadow-mold",
          )}
        >
          <span className="hud-sm text-ink-soft">{String(i + 1).padStart(2, "0")}</span>
          {item.label}
        </Link>
      ))}
    </nav>
  );
}
