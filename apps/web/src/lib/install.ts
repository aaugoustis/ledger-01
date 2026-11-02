// Suggesting the PWA install outside the installed app. Chromium browsers fire
// `beforeinstallprompt`, which lets a button open the install dialog; iOS has no such API, so it
// gets instructions instead. Other browsers (Firefox desktop) get nothing.

/** Chromium's install prompt event (not in the DOM typings). */
export interface BeforeInstallPromptEvent extends Event {
  prompt(): Promise<void>;
  userChoice: Promise<{ outcome: "accepted" | "dismissed" }>;
}

export type InstallMode = "none" | "prompt" | "ios";

export function isIos(userAgent: string, maxTouchPoints: number) {
  // iPadOS reports a desktop Mac user agent; touch points give it away.
  return /iPhone|iPad|iPod/.test(userAgent) || (/Macintosh/.test(userAgent) && maxTouchPoints > 1);
}

/** What to offer: nothing when already installed, a button when the browser can prompt. */
export function installMode(opts: {
  standalone: boolean;
  canPrompt: boolean;
  userAgent: string;
  maxTouchPoints: number;
}): InstallMode {
  if (opts.standalone) return "none";
  if (opts.canPrompt) return "prompt";
  if (isIos(opts.userAgent, opts.maxTouchPoints)) return "ios";
  return "none";
}

/** Whether this page runs as the installed app. */
export function runningStandalone() {
  return (
    window.matchMedia("(display-mode: standalone)").matches ||
    (navigator as Navigator & { standalone?: boolean }).standalone === true
  );
}

const DISMISSED = "ledger.install-dismissed";

export function installDismissed() {
  try {
    return window.localStorage.getItem(DISMISSED) === "1";
  } catch {
    return false;
  }
}

export function dismissInstall() {
  try {
    window.localStorage.setItem(DISMISSED, "1");
  } catch {
    // Private mode or blocked storage: it just shows again next time.
  }
}
