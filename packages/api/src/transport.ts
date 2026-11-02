import { createConnectTransport } from "@connectrpc/connect-web";

// The browser talks to the API through the web app's own origin: Next.js rewrites /api/* to the
// Go server locally, CloudFront routes it to API Gateway in AWS. Same origin keeps the session
// cookie first-party.
export function browserTransport(baseUrl = "/api") {
  return createConnectTransport({ baseUrl, useBinaryFormat: false });
}
