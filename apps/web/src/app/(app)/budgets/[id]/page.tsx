import type { Metadata } from "next";
import { BudgetDetail } from "./budget-detail";

export const metadata: Metadata = { title: "Budget" };

export default async function BudgetPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <BudgetDetail budgetId={id} />;
}
