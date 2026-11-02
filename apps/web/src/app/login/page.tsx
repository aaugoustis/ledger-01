import { buttonVariants } from "@ledger-01/ui/components/button";
import { Grill, HudLabel, StatusLed, TickRuler } from "@ledger-01/ui/components/hud";
import { Panel } from "@ledger-01/ui/components/panel";
import { cn } from "@ledger-01/ui/lib/utils";
import type { Metadata } from "next";
import { InstallSuggestion } from "./install-suggestion";

export const metadata: Metadata = { title: "Sign in" };

const errors: Record<string, string> = {
  not_allowed: "That Google account isn't on this ledger's allowlist.",
  cancelled: "Sign-in was cancelled.",
  exchange_failed: "Google sign-in failed. Try again.",
  invalid_token: "Google sign-in couldn't be verified. Try again.",
  email_unverified: "Your Google email address isn't verified.",
};

export default async function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ next?: string; error?: string }>;
}) {
  const { next, error } = await searchParams;
  const start = `/auth/google/start${next ? `?next=${encodeURIComponent(next)}` : ""}`;

  return (
    <main className="mx-auto flex min-h-dvh max-w-content-max flex-col justify-center px-4 py-12 sm:px-12">
      <Panel
        notch="lg"
        serial="UNIT-01 · REV.2026"
        hinge="48px"
        cross
        className="mx-auto w-full max-w-lg"
      >
        <div className="flex flex-col gap-6">
          <HudLabel code="S-00">Sign in</HudLabel>
          <h1 className="font-display text-display-md text-ink sm:text-display-lg">Ledger/01</h1>
          <p className="text-body-lg text-ink-soft">
            A private ledger for your household's wallets.
          </p>
          {error && (
            <p role="alert" className="rounded-md bg-danger-tint px-4 py-3 text-body-sm text-ink">
              {errors[error] ?? "Sign-in failed. Try again."}
            </p>
          )}
          <a href={start} className={cn(buttonVariants({ size: "lg" }), "self-start")}>
            Continue with Google
          </a>
          <div className="flex items-center justify-between gap-4">
            <StatusLed>Private ledger</StatusLed>
            <Grill count={4} />
          </div>
        </div>
      </Panel>
      <InstallSuggestion />
      <TickRuler
        className="mx-auto mt-12 w-full max-w-lg"
        labels={["00", "10", "20", "30", "40"]}
      />
    </main>
  );
}
