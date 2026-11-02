import type { Metadata } from "next";
import { ImportView } from "./import-view";

export const metadata: Metadata = { title: "Import" };

export default async function WalletImportPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <ImportView walletId={id} />;
}
