import { categoryColorVar, categoryIcon } from "@ledger-01/ui/lib/category-icons";
import { cn } from "@ledger-01/ui/lib/utils";

const sizes = {
  sm: { box: "size-7", icon: "size-3.5" },
  md: { box: "size-9", icon: "size-4.5" },
  lg: { box: "size-12", icon: "size-6" },
};

/**
 * A category's Lucide glyph in ink on a palette-colored disc. For transfers pass
 * icon="arrow-left-right" and color="cat-slate".
 */
export function CategoryIcon({
  icon,
  color,
  size = "md",
  label,
  className,
}: {
  icon: string;
  color: string;
  size?: keyof typeof sizes;
  /** Accessible name; omit when the category name is shown next to it. */
  label?: string;
  className?: string;
}) {
  const Icon = categoryIcon(icon);
  const s = sizes[size];
  // Named image when labelled; otherwise decorative (the category name sits next to it).
  const a11y = label
    ? { role: "img" as const, "aria-label": label }
    : { "aria-hidden": true as const };
  return (
    <span
      {...a11y}
      className={cn(
        "inline-flex shrink-0 items-center justify-center rounded-full text-ink shadow-[inset_0_1px_0_rgba(255,255,255,0.45),inset_0_-2px_0_rgba(28,30,33,0.08)]",
        s.box,
        className,
      )}
      style={{ background: categoryColorVar(color) }}
    >
      <Icon className={s.icon} strokeWidth={1.25} />
    </span>
  );
}

/** Small colored dot for labels and legends. */
export function ColorDot({ color, className }: { color: string; className?: string }) {
  return (
    <span
      aria-hidden
      className={cn("inline-block size-2.5 shrink-0 rounded-full", className)}
      style={{ background: categoryColorVar(color) }}
    />
  );
}
