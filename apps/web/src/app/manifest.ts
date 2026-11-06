import type { MetadataRoute } from "next";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "Ledger/01",
    short_name: "Ledger",
    description: "Personal and household finance ledger.",
    start_url: "/dashboard",
    scope: "/",
    display: "standalone",
    background_color: "#f3f4f6",
    theme_color: "#f3f4f6",
    icons: [
      { src: "/icon/192", sizes: "192x192", type: "image/png", purpose: "any" },
      { src: "/icon/512", sizes: "512x512", type: "image/png", purpose: "any" },
      { src: "/icon/512", sizes: "512x512", type: "image/png", purpose: "maskable" },
    ],
    // Press and hold the installed app's icon. The dashboard opens the form (components/quick-add).
    shortcuts: [
      {
        name: "New transaction",
        short_name: "New",
        description: "Add an expense, income or transfer",
        url: "/dashboard?new=transaction",
        icons: [{ src: "/icon/192", sizes: "192x192", type: "image/png" }],
      },
    ],
  };
}
