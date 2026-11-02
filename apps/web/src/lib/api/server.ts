import "server-only";

import type { DescService } from "@bufbuild/protobuf";
import { type Client, Code, ConnectError, createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-node";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

const baseUrl = (process.env.API_INTERNAL_URL ?? "http://127.0.0.1:8080").replace(/\/$/, "");

/**
 * A Connect client for server components and route handlers. It calls the Go API directly and
 * forwards the visitor's cookies so the API sees the same session as the browser.
 */
export async function serverClient<T extends DescService>(service: T): Promise<Client<T>> {
  const cookieHeader = (await cookies()).toString();
  const transport = createConnectTransport({
    baseUrl,
    httpVersion: "1.1",
    interceptors: [
      (next) => (req) => {
        if (cookieHeader) req.header.set("cookie", cookieHeader);
        return next(req);
      },
    ],
  });
  return createClient(service, transport);
}

/** Runs an API call, sending the visitor to /login when their session is missing or expired. */
export async function orLogin<T>(call: Promise<T>, next?: string): Promise<T> {
  try {
    return await call;
  } catch (err) {
    if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
      redirect(next ? `/login?next=${encodeURIComponent(next)}` : "/login");
    }
    throw err;
  }
}
