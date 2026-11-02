"use client";

import { Button } from "@ledger-01/ui/components/button";
import { Panel } from "@ledger-01/ui/components/panel";
import { Download, Share, X } from "lucide-react";
import { useEffect, useState } from "react";
import {
  type BeforeInstallPromptEvent,
  dismissInstall,
  type InstallMode,
  installDismissed,
  installMode,
  runningStandalone,
} from "@/lib/install";

/** Suggests installing Ledger/01 as an app when the login page runs in a browser tab. */
export function InstallSuggestion() {
  const [prompt, setPrompt] = useState<BeforeInstallPromptEvent>();
  const [mode, setMode] = useState<InstallMode>("none");

  useEffect(() => {
    if (installDismissed()) return;
    const decide = (canPrompt: boolean) =>
      setMode(
        installMode({
          standalone: runningStandalone(),
          canPrompt,
          userAgent: navigator.userAgent,
          maxTouchPoints: navigator.maxTouchPoints,
        }),
      );
    decide(false);
    const onPrompt = (e: Event) => {
      // Keep the browser's own mini-infobar away; the button below opens the dialog instead.
      e.preventDefault();
      setPrompt(e as BeforeInstallPromptEvent);
      decide(true);
    };
    const onInstalled = () => setMode("none");
    window.addEventListener("beforeinstallprompt", onPrompt);
    window.addEventListener("appinstalled", onInstalled);
    return () => {
      window.removeEventListener("beforeinstallprompt", onPrompt);
      window.removeEventListener("appinstalled", onInstalled);
    };
  }, []);

  if (mode === "none") return null;

  const dismiss = () => {
    dismissInstall();
    setMode("none");
  };

  return (
    <Panel recess notch="sm" className="mx-auto mt-6 w-full max-w-lg">
      <div className="flex items-start gap-3">
        <div className="flex min-w-0 flex-1 flex-col gap-3">
          <p className="text-body-md text-ink">Install Ledger/01 as an app</p>
          {mode === "prompt" ? (
            <>
              <p className="text-body-sm text-ink-soft">
                It opens in its own window and stays one tap away on your home screen.
              </p>
              <Button
                variant="secondary"
                size="sm"
                className="self-start"
                onClick={async () => {
                  if (!prompt) return;
                  await prompt.prompt();
                  await prompt.userChoice;
                  // A prompt event works once. If they said no, the browser fires a new one when
                  // it's willing to ask again, which brings this back.
                  setPrompt(undefined);
                  setMode("none");
                }}
              >
                <Download strokeWidth={1.25} /> Install app
              </Button>
            </>
          ) : (
            <p className="text-body-sm text-ink-soft">
              In Safari, tap{" "}
              <Share strokeWidth={1.25} className="inline size-4 align-text-bottom" aria-hidden />{" "}
              Share, then <strong className="text-ink">Add to Home Screen</strong>.
            </p>
          )}
        </div>
        <Button
          variant="ghost"
          size="icon-sm"
          aria-label="Don't suggest installing"
          onClick={dismiss}
        >
          <X strokeWidth={1.25} />
        </Button>
      </div>
    </Panel>
  );
}
