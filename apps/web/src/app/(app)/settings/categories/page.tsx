import type { Metadata } from "next";
import { CategoriesSettings } from "./categories-settings";

export const metadata: Metadata = { title: "Categories" };

export default function CategoriesPage() {
  return <CategoriesSettings />;
}
