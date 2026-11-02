import { type ClassValue, clsx } from "clsx";
import { extendTailwindMerge } from "tailwind-merge";

// Teach tailwind-merge the Module/01 text styles so `text-body-sm` and `text-ink` don't collide.
const twMerge = extendTailwindMerge({
  extend: {
    classGroups: {
      "font-size": [
        {
          text: [
            "display-xl",
            "display-lg",
            "display-md",
            "display-sm",
            "body-lg",
            "body-md",
            "body-sm",
            "label",
            "hud",
            "hud-sm",
            "hud-num",
            "code",
            "amount-xl",
            "amount-lg",
            "amount-md",
          ],
        },
      ],
    },
  },
});

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
