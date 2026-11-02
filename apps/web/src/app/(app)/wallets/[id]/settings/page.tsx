import { WalletSettings } from "./wallet-settings";

export default async function WalletSettingsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <WalletSettings walletId={id} />;
}
