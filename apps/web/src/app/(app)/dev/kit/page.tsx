import { notFound } from "next/navigation";
import { Kit } from "./kit";

// Kitchen sink of every UI component, for design review. Development builds only.
export default function KitPage() {
  if (process.env.NODE_ENV === "production") notFound();
  return <Kit />;
}
