import { CategoryReportView } from "./category-report";

export default async function CategoryReportPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <CategoryReportView categoryId={id} />;
}
