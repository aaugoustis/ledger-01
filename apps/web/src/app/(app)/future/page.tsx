import type { Metadata } from "next";
import { FutureView } from "./future-view";

export const metadata: Metadata = { title: "Future" };

export default function FuturePage() {
  return <FutureView />;
}
