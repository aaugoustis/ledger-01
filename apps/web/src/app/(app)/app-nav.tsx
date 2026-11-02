"use client";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@ledger-01/ui/components/dropdown-menu";
import { NavBar } from "@ledger-01/ui/components/nav-bar";
import { LogOut, Settings, Tags, User } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";

export function AppNav({
  name,
  email,
  avatarUrl,
}: {
  name: string;
  email: string;
  avatarUrl: string;
}) {
  const path = usePathname();
  const initials = (name || email).slice(0, 1).toUpperCase();

  async function signOut() {
    await fetch("/auth/logout", { method: "POST" });
    window.location.href = "/login";
  }

  return (
    <NavBar
      brand={
        <Link href="/dashboard" className="outline-none focus-visible:underline">
          <span className="sm:hidden">L/01</span>
          <span className="hidden sm:inline">Ledger/01</span>
        </Link>
      }
      serial="UNIT-01 · REV.2026"
      links={[
        // On phones the brand is the way home: four links and the menu don't fit at 360px.
        {
          label: "Dashboard",
          href: "/dashboard",
          active: path.startsWith("/dashboard"),
          desktopOnly: true,
        },
        { label: "Budgets", href: "/budgets", active: path.startsWith("/budgets") },
        {
          label: "Wallets",
          href: "/wallets",
          active: path.startsWith("/wallets") || path.startsWith("/future"),
        },
        {
          label: "Settings",
          href: "/settings/categories",
          active: path.startsWith("/settings"),
          desktopOnly: true,
        },
      ]}
      renderLink={(item, props) => <Link href={item.href} {...props} />}
      end={
        <DropdownMenu>
          <DropdownMenuTrigger
            aria-label="Account menu"
            className="flex size-11 shrink-0 items-center justify-center overflow-hidden rounded-full bg-shell text-label text-ink shadow-mold outline-none focus-visible:ring-[3px] focus-visible:ring-ring/60"
          >
            {avatarUrl ? (
              // biome-ignore lint/performance/noImgElement: tiny remote avatar, no optimisation needed
              <img
                src={avatarUrl}
                alt=""
                className="size-full object-cover"
                referrerPolicy="no-referrer"
              />
            ) : (
              initials
            )}
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56">
            <DropdownMenuLabel className="flex flex-col">
              <span className="text-label text-ink">{name || "Account"}</span>
              <span className="truncate text-body-sm text-ink-soft">{email}</span>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem asChild>
              <Link href="/settings/account">
                <User strokeWidth={1.25} /> Account
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild>
              <Link href="/settings/categories">
                <Settings strokeWidth={1.25} /> Categories
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild>
              <Link href="/settings/labels">
                <Tags strokeWidth={1.25} /> Labels
              </Link>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={signOut}>
              <LogOut strokeWidth={1.25} /> Sign out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      }
    />
  );
}
