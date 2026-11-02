import type { Metadata } from "next";
import { FutureView } from "@/app/(app)/future/future-view";

export const metadata: Metadata = { title: "Future" };

export default async function WalletFuturePage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <FutureView walletId={id} />;
}
