import type { Metadata } from "next";
import { WalletOverview } from "./wallet-overview";

export const metadata: Metadata = { title: "Overview" };

export default async function WalletOverviewPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <WalletOverview walletId={id} />;
}
