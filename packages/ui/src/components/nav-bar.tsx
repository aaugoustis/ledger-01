import { cn } from "@ledger-01/ui/lib/utils";
import type * as React from "react";
import { Fragment } from "react";

export type NavItem = {
  label: string;
  href: string;
  active?: boolean;
  /** Hidden on phones, e.g. a link the user menu also offers. */
  desktopOnly?: boolean;
};

/**
 * Floating frosted stadium bar: brand on the left, numbered links, an end slot (user menu).
 * `renderLink` lets the app use its router's Link component.
 */
export function NavBar({
  brand,
  serial,
  links,
  end,
  renderLink = (item, props) => <a href={item.href} {...props} />,
  className,
}: {
  brand: React.ReactNode;
  serial?: React.ReactNode;
  links: NavItem[];
  end?: React.ReactNode;
  renderLink?: (item: NavItem, props: React.ComponentProps<"a">) => React.ReactNode;
  className?: string;
}) {
  return (
    <nav
      aria-label="Main"
      className={cn(
        "glass flex h-nav-height items-center gap-2 rounded-full pr-2.5 pl-4 sm:gap-6 sm:pl-6",
        className,
      )}
    >
      <div className="flex shrink-0 flex-col gap-0.5">
        <span className="font-display text-display-sm whitespace-nowrap text-ink">{brand}</span>
        {serial !== undefined && (
          <span className="hud-sm hidden text-ink-soft sm:block">{serial}</span>
        )}
      </div>
      <div className="ml-auto flex min-w-0 items-center sm:gap-1">
        {links.map((item, i) => (
          <Fragment key={item.href}>
            {renderLink(item, {
              "aria-current": item.active ? "page" : undefined,
              className: cn(
                "inline-flex items-center gap-1.5 rounded-full px-2 py-2 text-label text-ink hover:bg-shell-recess sm:px-3.5",
                item.active && "text-signal-ink",
                item.desktopOnly && "max-sm:hidden",
              ),
              children: (
                <>
                  <span className="hud-sm hidden text-ink-soft md:inline">
                    {String(i + 1).padStart(2, "0")}
                  </span>
                  {item.label}
                </>
              ),
            })}
          </Fragment>
        ))}
      </div>
      {end}
    </nav>
  );
}
