import type { Metadata } from "next";
import { BudgetList } from "./budget-list";

export const metadata: Metadata = { title: "Budgets" };

export default function BudgetsPage() {
  return <BudgetList />;
}
