"use client";

import { timestampDate } from "@bufbuild/protobuf/wkt";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { OAuthService } from "@ledger-01/api";
import { Button } from "@ledger-01/ui/components/button";
import { Panel } from "@ledger-01/ui/components/panel";
import { Skeleton } from "@ledger-01/ui/components/skeleton";
import { Unplug } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";

const dateFormat = new Intl.DateTimeFormat("en", { dateStyle: "medium" });

/** Apps such as Claude that can read the ledger through MCP, and how to connect one. */
export default function ConnectedAppsPage() {
  const { data, isPending } = useQuery(OAuthService.method.listConnections, {});
  const revoke = useMutation(OAuthService.method.revokeConnection);
  const connections = data?.connections ?? [];
  // The MCP address is this site's own; known only in the browser.
  const [mcpUrl, setMcpUrl] = useState("");
  useEffect(() => setMcpUrl(`${window.location.origin}/mcp`), []);

  return (
    <div className="flex flex-col gap-6">
      <Panel serial="APP-01 · CONNECT" className="max-w-2xl">
        <div className="flex flex-col gap-3">
          <p className="text-body-sm text-ink">
            Let an AI assistant answer questions about your money, like “Am I spending more on
            restaurants than last month?”. In Claude, add a custom connector with this address, then
            sign in and allow read access.
          </p>
          <code className="self-start rounded-md bg-shell-recess px-3 py-2 font-mono text-body-sm text-ink break-all">
            {mcpUrl || "…/mcp"}
          </code>
          <p className="text-body-sm text-ink-soft">
            Connected apps can read everything in your ledger but can't change it.
          </p>
        </div>
      </Panel>

      <Panel serial="APP-02 · CONNECTED" className="max-w-2xl">
        {isPending ? (
          <Skeleton className="h-10" />
        ) : connections.length === 0 ? (
          <p className="text-body-sm text-ink-soft">No apps are connected.</p>
        ) : (
          <ul className="flex flex-col">
            {connections.map((c) => (
              <li
                key={c.id}
                className="flex items-center gap-3 border-b border-seam/60 py-3 last:border-b-0"
              >
                <div className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate text-body-md text-ink">{c.clientName}</span>
                  <span className="truncate text-body-sm text-ink-soft">
                    {c.redirectHost} · connected{" "}
                    {c.createdAt ? dateFormat.format(timestampDate(c.createdAt)) : ""}
                    {c.lastUsedAt
                      ? ` · last used ${dateFormat.format(timestampDate(c.lastUsedAt))}`
                      : " · not used yet"}
                  </span>
                </div>
                <Button
                  variant="secondary"
                  size="sm"
                  aria-label={`Disconnect ${c.clientName}`}
                  disabled={revoke.isPending}
                  onClick={async () => {
                    await revoke.mutateAsync({ id: c.id });
                    toast.success(`Disconnected ${c.clientName}`);
                  }}
                >
                  <Unplug strokeWidth={1.25} /> Disconnect
                </Button>
              </li>
            ))}
          </ul>
        )}
      </Panel>
    </div>
  );
}
