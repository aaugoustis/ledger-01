"use client";

import { Code, ConnectError } from "@connectrpc/connect";
import { TransportProvider } from "@connectrpc/connect-query";
import { browserTransport } from "@ledger-01/api";
import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";

function onError(err: unknown) {
  if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
    window.location.href = `/login?next=${encodeURIComponent(window.location.pathname + window.location.search)}`;
  }
}

function makeQueryClient(): QueryClient {
  const client: QueryClient = new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        retry: (count, err) =>
          !(err instanceof ConnectError && err.code !== Code.Unavailable) && count < 2,
      },
    },
    queryCache: new QueryCache({ onError }),
    mutationCache: new MutationCache({
      onError: (err) => {
        onError(err);
        toast.error(err instanceof ConnectError ? err.rawMessage : "Something went wrong");
      },
      // Every write can change balances, counts and lists, so refresh everything.
      onSuccess: async () => {
        await client.invalidateQueries();
      },
    }),
  });
  return client;
}

export function ApiProvider({ children }: { children: React.ReactNode }) {
  const [transport] = useState(() => browserTransport());
  const [queryClient] = useState(makeQueryClient);
  return (
    <TransportProvider transport={transport}>
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    </TransportProvider>
  );
}
