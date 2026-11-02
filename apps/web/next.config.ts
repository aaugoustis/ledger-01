import type { NextConfig } from "next";

// Locally the Go API runs on its own port. The browser only ever talks to this origin: /api/* and
// /auth/* are proxied to the API so the session cookie stays first-party, and so are the MCP
// endpoint and its OAuth server (/mcp, /oauth/*, /.well-known/oauth-*). In AWS, CloudFront does
// the same routing in front of API Gateway.
const api = (process.env.API_INTERNAL_URL ?? "http://127.0.0.1:8080").replace(/\/$/, "");

const config: NextConfig = {
  reactStrictMode: true,
  transpilePackages: ["@ledger-01/ui", "@ledger-01/api"],
  poweredByHeader: false,
  async rewrites() {
    return [
      { source: "/api/:path*", destination: `${api}/:path*` },
      { source: "/auth/:path*", destination: `${api}/auth/:path*` },
      { source: "/mcp", destination: `${api}/mcp` },
      { source: "/oauth/:path*", destination: `${api}/oauth/:path*` },
      { source: "/.well-known/:path(oauth-.*)", destination: `${api}/.well-known/:path` },
    ];
  },
  async headers() {
    // The consent page must never be framed (clickjacking an "Allow" click).
    return [
      {
        source: "/connect",
        headers: [
          { key: "Content-Security-Policy", value: "frame-ancestors 'none'" },
          { key: "X-Frame-Options", value: "DENY" },
        ],
      },
    ];
  },
};

export default config;
