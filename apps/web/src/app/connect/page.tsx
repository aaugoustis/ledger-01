import { Code, ConnectError } from "@connectrpc/connect";
import { OAuthService } from "@ledger-01/api";
import { HudLabel } from "@ledger-01/ui/components/hud";
import { Panel } from "@ledger-01/ui/components/panel";
import type { Metadata } from "next";
import { ApiProvider } from "@/lib/api/providers";
import { orLogin, serverClient } from "@/lib/api/server";
import { ConsentForm } from "./consent-form";

export const metadata: Metadata = { title: "Connect an app" };

/**
 * The consent page an MCP client (such as Claude) sends the user to from /oauth/authorize. The
 * request id comes from the API's redirect; approving returns the user to the client with a code.
 */
export default async function ConnectPage({
  searchParams,
}: {
  searchParams: Promise<{ request?: string }>;
}) {
  const { request: id = "" } = await searchParams;
  const oauth = await serverClient(OAuthService);
  const request = await orLogin(
    oauth.getAuthorizationRequest({ id }).then(
      (r) => r.request,
      (err) => {
        // Expired, already answered or made up: the app has to start over.
        if (
          err instanceof ConnectError &&
          [Code.NotFound, Code.InvalidArgument].includes(err.code)
        ) {
          return undefined;
        }
        throw err;
      },
    ),
    `/connect?request=${encodeURIComponent(id)}`,
  );

  return (
    <main className="mx-auto flex min-h-dvh max-w-content-max flex-col justify-center px-4 py-12 sm:px-12">
      <Panel notch="lg" serial="LINK-01" hinge="48px" cross className="mx-auto w-full max-w-lg">
        <div className="flex flex-col gap-6">
          <HudLabel code="S-05">Connect</HudLabel>
          {request ? (
            <ApiProvider>
              <ConsentForm
                id={request.id}
                clientName={request.clientName}
                redirectHost={request.redirectHost}
              />
            </ApiProvider>
          ) : (
            <>
              <h1 className="font-display text-display-sm text-ink">This link has expired</h1>
              <p className="text-body-md text-ink-soft">
                Connection requests last 10 minutes and work once. Start again from the app you were
                connecting.
              </p>
            </>
          )}
        </div>
      </Panel>
    </main>
  );
}
