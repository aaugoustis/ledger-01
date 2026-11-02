import { WalletView } from "./wallet-view";

export default async function WalletPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <WalletView walletId={id} />;
}
