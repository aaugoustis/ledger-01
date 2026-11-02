import { SectionHeader } from "@ledger-01/ui/components/hud";
import { SettingsNav } from "./settings-nav";

export default function SettingsLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-10">
      <SectionHeader code="S-04" label="Settings" title="Settings" ruler />
      <div className="grid gap-8 md:grid-cols-[200px_1fr]">
        <SettingsNav />
        <div className="min-w-0">{children}</div>
      </div>
    </div>
  );
}
