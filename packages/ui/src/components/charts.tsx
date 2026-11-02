"use client";

import { categoryColorVar } from "@ledger-01/ui/lib/category-icons";
import { formatMoney } from "@ledger-01/ui/lib/money";
import { cn } from "@ledger-01/ui/lib/utils";
import { useEffect, useRef, useState } from "react";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Pie,
  PieChart,
  ReferenceLine,
  XAxis,
  YAxis,
} from "recharts";
import { CategoryIcon } from "./category-icon";
import { type ChartConfig, ChartContainer, ChartTooltip } from "./chart";

// Module/01 charts. Marks are thin, axes and grid recede (seam / hud-mark), text uses ink tokens,
// and series colors come from tokens: chart-line (single series), chart-income / chart-expense
// (paired with position above/below the baseline), category palette for categories.

type Money = { currency: string; minorUnits: number };

const toNumber = (minor: bigint, units: number) => Number(minor) / 10 ** units;

const axisProps = {
  tickLine: false,
  axisLine: false,
  tick: { fill: "var(--color-ink-soft)", fontSize: 11, fontFamily: "var(--font-mono)" },
} as const;

// Enough digits that neighbouring ticks never read the same (11.38K, 11.42K).
const compactFormat = new Intl.NumberFormat("en-US", {
  notation: "compact",
  maximumSignificantDigits: 4,
});

function compact(n: number) {
  return compactFormat.format(n);
}

function TooltipCard({
  title,
  rows,
}: {
  title: string;
  rows: { label: string; value: string; swatch?: string }[];
}) {
  return (
    <div className="min-w-44 rounded-md border border-seam bg-shell px-3 py-2 shadow-float">
      <div className="hud-sm mb-1.5 text-ink-soft">{title}</div>
      {rows.map((r) => (
        <div key={r.label} className="flex items-center justify-between gap-4 text-body-sm">
          <span className="flex items-center gap-1.5 text-ink-soft">
            {r.swatch && <span className="size-2 rounded-full" style={{ background: r.swatch }} />}
            {r.label}
          </span>
          <span className="tabular text-ink">{r.value}</span>
        </div>
      ))}
    </div>
  );
}

export type BalancePoint = { label: string; tooltipLabel: string; balance: bigint };

/** One series over time: the balance line with a soft area and a crosshair tooltip. */
export function BalanceChart({
  points,
  money,
  className,
}: {
  points: BalancePoint[];
  money: Money;
  className?: string;
}) {
  const data = points.map((p) => ({ ...p, value: toNumber(p.balance, money.minorUnits) }));
  const config = {
    value: { label: "Balance", color: "var(--color-chart-line)" },
  } satisfies ChartConfig;
  return (
    <ChartContainer config={config} className={cn("aspect-auto h-56 w-full", className)}>
      <AreaChart data={data} margin={{ left: 4, right: 20, top: 8, bottom: 0 }}>
        <CartesianGrid vertical={false} stroke="var(--color-seam)" strokeDasharray="2 4" />
        <XAxis dataKey="label" {...axisProps} minTickGap={24} />
        <YAxis {...axisProps} width={48} tickFormatter={compact} domain={["auto", "auto"]} />
        <ChartTooltip
          cursor={{ stroke: "var(--color-hud-mark)", strokeWidth: 1 }}
          content={({ active, payload }) => {
            const p = active && payload?.[0]?.payload;
            if (!p) return null;
            return (
              <TooltipCard
                title={p.tooltipLabel}
                rows={[
                  {
                    label: "Balance",
                    value: formatMoney(p.balance, money.currency, money.minorUnits, {
                      sign: "always",
                    }),
                  },
                ]}
              />
            );
          }}
        />
        <Area
          dataKey="value"
          type="linear"
          stroke="var(--color-chart-line)"
          strokeWidth={2}
          fill="var(--color-signal-tint)"
          fillOpacity={0.6}
          dot={false}
          activeDot={{
            r: 4,
            fill: "var(--color-chart-line)",
            stroke: "var(--color-shell)",
            strokeWidth: 2,
          }}
          isAnimationActive={false}
        />
      </AreaChart>
    </ChartContainer>
  );
}

export type FlowPoint = { label: string; tooltipLabel: string; income: bigint; expense: bigint };

/**
 * Income above the baseline, expenses below it, per bucket. With `only`, a single series (no
 * legend: the panel title names it).
 */
export function FlowChart({
  points,
  money,
  only,
  className,
}: {
  points: FlowPoint[];
  money: Money;
  only?: "income" | "expense";
  className?: string;
}) {
  const showIncome = only !== "expense";
  const showExpense = only !== "income";
  const data = points.map((p) => ({
    ...p,
    incomeValue: toNumber(p.income, money.minorUnits),
    expenseValue: -toNumber(p.expense, money.minorUnits),
  }));
  const config = {
    incomeValue: { label: "Income", color: "var(--color-chart-income)" },
    expenseValue: { label: "Expenses", color: "var(--color-chart-expense)" },
  } satisfies ChartConfig;
  return (
    <div className={cn("flex flex-col gap-2", className)}>
      <ChartContainer config={config} className="aspect-auto h-56 w-full">
        <BarChart
          data={data}
          stackOffset="sign"
          margin={{ left: 4, right: 20, top: 8, bottom: 0 }}
          barCategoryGap="20%"
        >
          <CartesianGrid vertical={false} stroke="var(--color-seam)" strokeDasharray="2 4" />
          <XAxis dataKey="label" {...axisProps} minTickGap={24} />
          <YAxis {...axisProps} width={48} tickFormatter={(v: number) => compact(Math.abs(v))} />
          <ReferenceLine y={0} stroke="var(--color-hud-mark)" />
          <ChartTooltip
            cursor={{ fill: "var(--color-shell-recess)", opacity: 0.6 }}
            content={({ active, payload }) => {
              const p = active && payload?.[0]?.payload;
              if (!p) return null;
              return (
                <TooltipCard
                  title={p.tooltipLabel}
                  rows={[
                    ...(showIncome
                      ? [
                          {
                            label: "Income",
                            swatch: "var(--color-chart-income)",
                            value: formatMoney(p.income, money.currency, money.minorUnits, {
                              sign: "always",
                            }),
                          },
                        ]
                      : []),
                    ...(showExpense
                      ? [
                          {
                            label: "Expenses",
                            swatch: "var(--color-chart-expense)",
                            value: formatMoney(
                              0n - (p.expense as bigint),
                              money.currency,
                              money.minorUnits,
                            ),
                          },
                        ]
                      : []),
                  ]}
                />
              );
            }}
          />
          {showIncome && (
            <Bar
              dataKey="incomeValue"
              stackId="flow"
              fill="var(--color-chart-income)"
              radius={[4, 4, 0, 0]}
              isAnimationActive={false}
            />
          )}
          {showExpense && (
            <Bar
              dataKey="expenseValue"
              stackId="flow"
              fill="var(--color-chart-expense)"
              radius={[0, 0, 4, 4]}
              isAnimationActive={false}
            />
          )}
        </BarChart>
      </ChartContainer>
      {!only && (
        <div className="flex gap-4 pl-14">
          {(["incomeValue", "expenseValue"] as const).map((k) => (
            <span key={k} className="hud-sm flex items-center gap-1.5 text-ink-soft">
              <span className="size-2 rounded-xs" style={{ background: config[k].color }} />
              {config[k].label}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

export type DonutSlice = {
  key: string;
  label: string;
  color: string;
  amount: bigint;
  /** Category icon for the slice's callout. Charts without icons get color dots. */
  icon?: string;
};

// Donut callouts, in px: a leader from the ring to a marker (the category icon, or a dot), then
// the share past the marker. Slices under 2% get none; the tooltip and the list still name them.
const callout = {
  leader: 12,
  icon: 28, // CategoryIcon "sm"
  dot: 12,
  gap: 5,
  charWidth: 7.8, // hud-sm: 10px mono plus tracking
  textHeight: 12,
  minShare: 50n, // amount * 50 >= total
};
const donutMaxRadius = 112;
// Below this the shares drop out first, so the total still fits in the hole.
const donutMinRadius = 56;
const TAU = 2 * Math.PI;

/** Angles run clockwise from 12 o'clock, like the donut's slices. */
function polar(cx: number, cy: number, r: number, angle: number) {
  return { x: cx + r * Math.sin(angle), y: cy - r * Math.cos(angle) };
}

/** Pushes sorted angles apart, the least it can, until neighbours are at least `min` apart. */
function spreadAngles(angles: number[], min: number): number[] {
  const p = [...angles];
  const n = p.length;
  if (n < 2) return p;
  const step = Math.min(min, TAU / n);
  for (let round = 0; round < 200; round++) {
    let moved = false;
    for (let i = 0; i < n; i++) {
      const j = (i + 1) % n;
      const a = p[i] ?? 0;
      const b = (p[j] ?? 0) + (j === 0 ? TAU : 0);
      if (b - a < step - 1e-6) {
        const push = (step - (b - a)) / 2;
        p[i] = a - push;
        p[j] = (p[j] ?? 0) + push;
        moved = true;
      }
    }
    if (!moved) break;
  }
  return p;
}

function useWidth() {
  const ref = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(0);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const observer = new ResizeObserver(([entry]) => {
      if (entry) setWidth(Math.floor(entry.contentRect.width));
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, []);
  return [ref, width] as const;
}

/**
 * Shares of a total, clockwise from 12 o'clock. Up to 7 slices plus "Other". Each slice of 2% or
 * more gets a callout outside the ring (its icon and share), so it reads without the legend; the
 * caller still shows the ranked list beside it.
 */
export function DonutChart({
  slices,
  money,
  centerLabel,
  className,
}: {
  slices: DonutSlice[];
  money: Money;
  centerLabel: string;
  className?: string;
}) {
  const [ref, width] = useWidth();
  const sorted = [...slices].sort((a, b) =>
    b.amount > a.amount ? 1 : b.amount < a.amount ? -1 : 0,
  );
  const withIcons = slices.some((s) => s.icon);
  const top = sorted.slice(0, 7);
  const rest = sorted.slice(7).reduce((s, x) => s + x.amount, 0n);
  if (rest > 0n) {
    top.push({
      key: "other",
      label: "Other",
      color: "cat-slate",
      amount: rest,
      icon: withIcons ? "ellipsis" : undefined,
    });
  }
  const total = sorted.reduce((s, x) => s + x.amount, 0n);
  const data = top.map((s) => ({ ...s, value: toNumber(s.amount, money.minorUnits) }));

  const sum = data.reduce((s, d) => s + d.value, 0);
  let acc = 0;
  const arcs = data.map((d) => {
    const from = sum > 0 ? (acc / sum) * TAU : 0;
    acc += d.value;
    const share = total > 0n ? Number((d.amount * 1000n) / total) / 10 : 0;
    return { ...d, from, to: sum > 0 ? (acc / sum) * TAU : 0, share: `${share.toFixed(1)}%` };
  });
  const labelled = arcs.filter((a) => a.amount * callout.minShare >= total && total > 0n);

  // The ring gets the room the callouts leave, up to its maximum size.
  const marker = withIcons ? callout.icon : callout.dot;
  const textWidth = Math.max(0, ...labelled.map((a) => a.share.length)) * callout.charWidth;
  const withShares = callout.leader + marker + callout.gap + textWidth;
  const showShares = width === 0 || width / 2 - withShares >= donutMinRadius;
  const marginX = showShares ? withShares : callout.leader + marker;
  const marginY = callout.leader + marker + (showShares ? callout.gap + callout.textHeight : 0);
  const radius =
    width === 0 ? donutMaxRadius : Math.max(32, Math.min(donutMaxRadius, width / 2 - marginX));
  const height = Math.round(2 * (radius + marginY));
  const cx = width / 2;
  const cy = height / 2;
  const inner = radius * 0.62;

  const orbit = radius + callout.leader + marker / 2;
  const clearance = Math.max(marker, showShares ? textWidth : 0) + 6;
  const angles = spreadAngles(
    labelled.map((a) => (a.from + a.to) / 2),
    2 * Math.asin(Math.min(1, clearance / 2 / orbit)),
  );
  const callouts = labelled.map((a, i) => {
    const angle = angles[i] ?? 0;
    // The leader leaves the ring inside its own slice, as close to the marker as it can.
    const pad = Math.min(0.05, (a.to - a.from) / 4);
    const start = polar(cx, cy, radius + 3, Math.min(a.to - pad, Math.max(a.from + pad, angle)));
    const center = polar(cx, cy, orbit, angle);
    const dx = start.x - center.x;
    const dy = start.y - center.y;
    const len = Math.hypot(dx, dy) || 1;
    const end = {
      x: center.x + (dx / len) * (marker / 2 + 1),
      y: center.y + (dy / len) * (marker / 2 + 1),
    };
    // The share sits past the marker, aligned away from the ring.
    const text = polar(cx, cy, orbit + marker / 2 + callout.gap, angle);
    const sx = Math.sin(angle);
    const sy = -Math.cos(angle);
    const tx = sx > 0.38 ? "0" : sx < -0.38 ? "-100%" : "-50%";
    const ty = sy > 0.38 ? "0" : sy < -0.38 ? "-100%" : "-50%";
    return { ...a, start, end, center, text, align: `translate(${tx}, ${ty})` };
  });

  const totalText = formatMoney(total, money.currency, money.minorUnits, { code: false });
  // Shrink the total until it fits the hole (mono digits are about 0.6em wide).
  const totalSize = Math.max(
    10,
    Math.min(15, Math.floor((inner * 2 * 0.84) / (totalText.length * 0.62))),
  );

  return (
    <div
      ref={ref}
      className={cn(
        "relative w-full [&_.recharts-layer]:outline-hidden [&_.recharts-sector]:outline-hidden [&_.recharts-surface]:outline-hidden",
        className,
      )}
      style={{ height }}
    >
      {width > 0 && (
        <>
          <div aria-hidden className="pointer-events-none absolute inset-0">
            <svg aria-hidden width={width} height={height} className="absolute inset-0">
              {callouts.map((c) => (
                <line
                  key={c.key}
                  x1={c.start.x}
                  y1={c.start.y}
                  x2={c.end.x}
                  y2={c.end.y}
                  stroke={categoryColorVar(c.color)}
                  strokeWidth={1.5}
                  strokeLinecap="round"
                />
              ))}
              <text
                x={cx}
                y={cy - 6}
                textAnchor="middle"
                className="hud-sm"
                fill="var(--color-ink-soft)"
              >
                {centerLabel}
              </text>
              <text
                x={cx}
                y={cy + 14}
                textAnchor="middle"
                className="tabular"
                fontSize={totalSize}
                fontWeight={700}
                fill="var(--color-ink)"
              >
                {totalText}
              </text>
            </svg>
            {callouts.map((c) => (
              <div key={c.key}>
                <span
                  className="absolute -translate-x-1/2 -translate-y-1/2"
                  style={{ left: c.center.x, top: c.center.y }}
                >
                  {withIcons ? (
                    <CategoryIcon
                      icon={c.icon ?? "circle-help"}
                      color={c.color}
                      size="sm"
                      className="flex"
                    />
                  ) : (
                    <span
                      className="block size-3 rounded-full"
                      style={{ background: categoryColorVar(c.color) }}
                    />
                  )}
                </span>
                {showShares && (
                  <span
                    className="hud-sm absolute whitespace-nowrap text-ink"
                    style={{ left: c.text.x, top: c.text.y, transform: c.align }}
                  >
                    {c.share}
                  </span>
                )}
              </div>
            ))}
          </div>
          <PieChart width={width} height={height} margin={{ top: 0, right: 0, bottom: 0, left: 0 }}>
            <ChartTooltip
              content={({ active, payload }) => {
                const p = active && payload?.[0]?.payload;
                if (!p) return null;
                const share = total > 0n ? Number((p.amount * 1000n) / total) / 10 : 0;
                return (
                  <TooltipCard
                    title={p.label}
                    rows={[
                      {
                        label: "Amount",
                        value: formatMoney(p.amount, money.currency, money.minorUnits),
                      },
                      { label: "Share", value: `${share.toFixed(1)}%` },
                    ]}
                  />
                );
              }}
            />
            <Pie
              data={data}
              dataKey="value"
              nameKey="label"
              cx={cx}
              cy={cy}
              innerRadius={inner}
              outerRadius={radius}
              startAngle={90}
              endAngle={-270}
              stroke={data.length > 1 ? "var(--color-shell)" : "none"}
              strokeWidth={2}
              isAnimationActive={false}
            >
              {data.map((s) => (
                <Cell key={s.key} fill={categoryColorVar(s.color)} />
              ))}
            </Pie>
          </PieChart>
        </>
      )}
    </div>
  );
}
