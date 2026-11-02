"use client";

import { Amount } from "@ledger-01/ui/components/amount";
import { Badge } from "@ledger-01/ui/components/badge";
import { Button } from "@ledger-01/ui/components/button";
import { CategoryIcon, ColorDot } from "@ledger-01/ui/components/category-icon";
import { Checkbox } from "@ledger-01/ui/components/checkbox";
import { Field } from "@ledger-01/ui/components/field";
import { FxConverter } from "@ledger-01/ui/components/fx-converter";
import {
  Crosshair,
  Grill,
  HudLabel,
  SectionHeader,
  StatusLed,
  TickRuler,
} from "@ledger-01/ui/components/hud";
import { Input } from "@ledger-01/ui/components/input";
import { MoneyInput } from "@ledger-01/ui/components/money-input";
import { GlassPanel, Panel } from "@ledger-01/ui/components/panel";
import { Skeleton } from "@ledger-01/ui/components/skeleton";
import { Tabs, TabsList, TabsTrigger } from "@ledger-01/ui/components/tabs";
import { categoryColors, categoryIcons } from "@ledger-01/ui/lib/category-icons";
import { type FxState, initialFxState } from "@ledger-01/ui/lib/fx";
import { Plus } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

const swatches = [
  "canvas",
  "shell",
  "shell-recess",
  "seam",
  "ink",
  "ink-soft",
  "hud-mark",
  "signal",
  "signal-ink",
  "signal-tint",
  "studio",
  "visor",
  "led-on",
  "income-ink",
  "expense-ink",
  "danger",
  "danger-tint",
];

export function Kit() {
  const [fx, setFx] = useState<FxState>(initialFxState());
  const icons = Object.keys(categoryIcons);

  return (
    <div className="flex flex-col gap-16">
      <SectionHeader code="K-00" label="Dev" title="Component kit" ruler />

      <section className="flex flex-col gap-4">
        <HudLabel code="K-01">Color tokens</HudLabel>
        <div className="grid grid-cols-3 gap-3 sm:grid-cols-6">
          {swatches.map((s) => (
            <div key={s} className="flex flex-col gap-1">
              <div
                className="h-12 rounded-md border border-seam"
                style={{ background: `var(--color-${s})` }}
              />
              <span className="hud-sm text-ink-soft">{s}</span>
            </div>
          ))}
        </div>
        <div className="flex flex-wrap gap-2">
          {categoryColors.map((c, i) => (
            <CategoryIcon key={c} color={c} icon={icons[i * 3] ?? "circle-help"} />
          ))}
        </div>
      </section>

      <section className="flex flex-col gap-4">
        <HudLabel code="K-02">Type</HudLabel>
        <p className="font-display text-display-lg">Display large</p>
        <p className="font-display text-display-sm">Display small</p>
        <p className="text-body-lg text-ink-soft">Body large for lead paragraphs.</p>
        <p className="text-body-md">Body medium for paragraphs and rows.</p>
        <p className="hud text-ink-soft">HUD label · S-02</p>
        <div className="flex flex-wrap gap-6">
          <Amount minor={123456n} currency="USD" minorUnits={2} size="xl" />
          <Amount minor={-248015n} currency="USD" minorUnits={2} size="lg" />
          <Amount minor={-1076n} currency="USD" minorUnits={2} code={false} />
          <Amount minor={0n} currency="UYU" minorUnits={2} />
        </div>
      </section>

      <section className="flex flex-col gap-4">
        <HudLabel code="K-03">Controls</HudLabel>
        <div className="flex flex-wrap items-center gap-3">
          <Button>
            <Plus strokeWidth={1.25} /> Primary
          </Button>
          <Button variant="secondary">Secondary</Button>
          <Button variant="outline">Outline</Button>
          <Button variant="ghost">Ghost</Button>
          <Button variant="destructive">Delete</Button>
          <Button variant="link">Link</Button>
          <Button onClick={() => toast.success("Saved")}>Toast</Button>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <Badge>Default</Badge>
          <Badge variant="signal">Signal</Badge>
          <Badge variant="tint">Tint</Badge>
          <Badge variant="outline">Outline</Badge>
          <Badge variant="destructive">Error</Badge>
          <StatusLed>Live</StatusLed>
          <StatusLed tone="signal">Pending</StatusLed>
          <StatusLed tone="off">Off</StatusLed>
          <span className="flex items-center gap-1 text-body-sm">
            <ColorDot color="cat-teal" /> Label
          </span>
        </div>
        <div className="grid max-w-3xl gap-4 sm:grid-cols-3">
          <Field label="Text input" htmlFor="k-in">
            <Input id="k-in" placeholder="Write a note" />
          </Field>
          <Field label="Money input" htmlFor="k-money">
            <MoneyInput id="k-money" currency="USD" />
          </Field>
          <Field label="Invalid" htmlFor="k-bad" error="Use up to 2 decimals">
            <Input id="k-bad" aria-invalid defaultValue="1.234" />
          </Field>
        </div>
        <div className="flex items-center gap-2">
          <Checkbox id="k-check" defaultChecked />
          <label htmlFor="k-check" className="text-body-sm">
            Keep open to add more
          </label>
        </div>
        <Tabs defaultValue="expense" className="max-w-md">
          <TabsList className="w-full">
            <TabsTrigger value="expense">Expense</TabsTrigger>
            <TabsTrigger value="income">Income</TabsTrigger>
            <TabsTrigger value="transfer">Transfer</TabsTrigger>
          </TabsList>
        </Tabs>
      </section>

      <section className="flex flex-col gap-4">
        <HudLabel code="K-04">Three-field converter</HudLabel>
        <FxConverter
          className="max-w-3xl"
          from={{ code: "USD", minorUnits: 2 }}
          to={{ code: "UYU", minorUnits: 2 }}
          value={fx}
          onChange={setFx}
        />
        <p className="hud-sm text-ink-soft">order (oldest first): {fx.order.join(" → ")}</p>
      </section>

      <section className="grid gap-6 sm:grid-cols-2">
        <Panel serial="PNL-01 · DEFAULT" hinge="32px" cross>
          <p className="text-body-md">Panel with serial, hinge tab and crosshair.</p>
        </Panel>
        <Panel recess notch="sm">
          <p className="text-body-md">Recessed panel, small notch.</p>
        </Panel>
        <div className="rounded-lg bg-studio p-6">
          <GlassPanel>
            <p className="text-body-md">Glass panel over the studio backdrop.</p>
          </GlassPanel>
        </div>
        <Panel>
          <div className="flex flex-col gap-4">
            <Grill />
            <TickRuler labels={["0", "10", "20", "30"]} />
            <Crosshair tone="signal" />
            <Skeleton className="h-6 w-40" />
          </div>
        </Panel>
      </section>
    </div>
  );
}
