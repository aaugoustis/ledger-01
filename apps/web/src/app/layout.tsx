import { Toaster } from "@ledger-01/ui/components/sonner";
import { TooltipProvider } from "@ledger-01/ui/components/tooltip";
import type { Metadata, Viewport } from "next";
import { Manrope, Michroma, Space_Mono } from "next/font/google";
import "./globals.css";

// Font variables match the stacks generated from tokens.json: var(--font-michroma, Michroma), …
const michroma = Michroma({ weight: "400", subsets: ["latin"], variable: "--font-michroma" });
const manrope = Manrope({ subsets: ["latin"], variable: "--font-manrope" });
const spaceMono = Space_Mono({
  weight: ["400", "700"],
  subsets: ["latin"],
  variable: "--font-space-mono",
});

export const metadata: Metadata = {
  title: { default: "Ledger/01", template: "%s · Ledger/01" },
  description: "Personal and household finance ledger.",
  applicationName: "Ledger/01",
  appleWebApp: { capable: true, title: "Ledger", statusBarStyle: "default" },
};

export const viewport: Viewport = {
  themeColor: "#f3f4f6",
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${michroma.variable} ${manrope.variable} ${spaceMono.variable}`}>
      <body className="min-h-dvh">
        <TooltipProvider delayDuration={300}>{children}</TooltipProvider>
        {/* On phones, toasts sit above the quick-add button instead of covering it. */}
        <Toaster
          position="bottom-center"
          mobileOffset={{ bottom: "calc(env(safe-area-inset-bottom) + 88px)" }}
        />
      </body>
    </html>
  );
}
