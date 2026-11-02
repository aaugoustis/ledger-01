import type { Metadata } from "next";
import { WalletGrid } from "./wallet-grid";

export const metadata: Metadata = { title: "Wallets" };

export default function WalletsPage() {
  return <WalletGrid />;
}
