import type { Member } from "@ledger-01/api";
import { cn } from "@ledger-01/ui/lib/utils";

/** First letters of a person's name (or email), for an avatar without a picture. */
export function initials(m: Pick<Member, "name" | "email">) {
  const words = (m.name || m.email).trim().split(/\s+/);
  return words
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase() ?? "")
    .join("");
}

/** A household member's round avatar: their Google picture, or initials. */
export function MemberAvatar({
  member,
  className,
}: {
  member: Pick<Member, "name" | "email" | "avatarUrl">;
  className?: string;
}) {
  return (
    <span
      title={member.name || member.email}
      className={cn(
        "flex size-8 shrink-0 items-center justify-center overflow-hidden rounded-full bg-shell-recess text-label text-ink",
        className,
      )}
    >
      {member.avatarUrl ? (
        // biome-ignore lint/performance/noImgElement: tiny remote avatar, no optimisation needed
        <img
          src={member.avatarUrl}
          alt=""
          className="size-full object-cover"
          referrerPolicy="no-referrer"
        />
      ) : (
        initials(member)
      )}
    </span>
  );
}
