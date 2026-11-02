import { type NextRequest, NextResponse } from "next/server";

// Cheap gate: without a session cookie, app pages redirect to /login. The API still validates
// the session on every call; this only avoids rendering an app shell for signed-out visitors.
export function proxy(req: NextRequest) {
  if (req.cookies.has("ledger_session")) return NextResponse.next();
  const url = req.nextUrl.clone();
  url.pathname = "/login";
  url.search = "";
  const next = req.nextUrl.pathname + req.nextUrl.search;
  if (next !== "/") url.searchParams.set("next", next);
  return NextResponse.redirect(url);
}

export const config = {
  // Everything except the login page, API/auth/MCP/OAuth proxies, Next internals and PWA files.
  matcher: [
    "/((?!login|api/|auth/|mcp$|oauth/|\\.well-known/|_next/|icon|apple-icon|manifest.webmanifest|favicon.ico).*)",
  ],
};
