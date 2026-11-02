"use client";

import { useMutation } from "@connectrpc/connect-query";
import { OAuthService } from "@ledger-01/api";
import { Button } from "@ledger-01/ui/components/button";
import { Check, X } from "lucide-react";
import { useState } from "react";

export function ConsentForm({
  id,
  clientName,
  redirectHost,
}: {
  id: string;
  clientName: string;
  redirectHost: string;
}) {
  const approve = useMutation(OAuthService.method.approveAuthorization);
  const deny = useMutation(OAuthService.method.denyAuthorization);
  // Stays true while the browser leaves for the app, so neither button can be pressed twice.
  const [leaving, setLeaving] = useState(false);

  async function answer(allow: boolean) {
    setLeaving(true);
    try {
      const res = allow ? await approve.mutateAsync({ id }) : await deny.mutateAsync({ id });
      window.location.assign(res.redirectUrl);
    } catch {
      setLeaving(false); // toasted
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <h1 className="font-display text-display-sm text-ink">Connect {clientName}?</h1>
      <p className="text-body-md text-ink">
        <strong>{clientName}</strong> wants read-only access to your ledger: wallets, transactions,
        categories, labels, reports and budgets. It can't change anything.
      </p>
      <p className="text-body-sm text-ink-soft">
        After you allow it, you go back to <strong className="text-ink">{redirectHost}</strong>.
        Only allow apps you started connecting yourself. You can disconnect it any time in Settings,
        Connected apps.
      </p>
      <div className="flex flex-wrap gap-3">
        <Button onClick={() => answer(true)} disabled={leaving}>
          <Check strokeWidth={1.25} /> Allow read access
        </Button>
        <Button variant="secondary" onClick={() => answer(false)} disabled={leaving}>
          <X strokeWidth={1.25} /> Deny
        </Button>
      </div>
    </div>
  );
}
