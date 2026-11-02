import type { ReactNode, MouseEventHandler, CSSProperties } from "react";

export interface ButtonProps { variant?: "primary" | "secondary" | "ghost"; href?: string; onClick?: MouseEventHandler; arrow?: boolean; disabled?: boolean; className?: string; children: ReactNode; }
export declare function Button(props: ButtonProps): JSX.Element;

export interface TagProps { tone?: "signal" | "tint" | "outline"; led?: boolean; className?: string; children: ReactNode; }
export declare function Tag(props: TagProps): JSX.Element;

export interface StatusLedProps { tone?: "signal" | "off"; className?: string; children: ReactNode; }
export declare function StatusLed(props: StatusLedProps): JSX.Element;

export interface HudLabelProps { code?: string; end?: ReactNode; rule?: boolean; className?: string; children: ReactNode; }
export declare function HudLabel(props: HudLabelProps): JSX.Element;

export interface TickRulerProps { labels?: string[]; className?: string; }
export declare function TickRuler(props: TickRulerProps): JSX.Element;

export interface CrosshairProps { tone?: "signal"; className?: string; style?: CSSProperties; }
export declare function Crosshair(props: CrosshairProps): JSX.Element;

export interface GrillProps { count?: number; slotWidth?: number; gap?: string; tone?: "signal" | "on-signal"; className?: string; }
export declare function Grill(props: GrillProps): JSX.Element;

export interface SectionHeaderProps { title: ReactNode; code?: string; label?: ReactNode; index?: string; aside?: ReactNode; ruler?: boolean; as?: "h1" | "h2" | "h3"; className?: string; }
export declare function SectionHeader(props: SectionHeaderProps): JSX.Element;

export interface PanelProps { serial?: string; notch?: "sm" | "md" | "lg" | false; recess?: boolean; cross?: boolean; style?: CSSProperties; className?: string; children: ReactNode; }
export declare function Panel(props: PanelProps): JSX.Element;

export interface GlassPanelProps { style?: CSSProperties; className?: string; children: ReactNode; }
export declare function GlassPanel(props: GlassPanelProps): JSX.Element;

export interface SpecItem { label: string; value: ReactNode; }
export interface SpecReadoutProps { items: SpecItem[]; className?: string; }
export declare function SpecReadout(props: SpecReadoutProps): JSX.Element;

export interface TerminalProps { lines: string[]; title?: string; className?: string; }
export declare function Terminal(props: TerminalProps): JSX.Element;

export interface NavItem { label: string; href?: string; active?: boolean; }
export interface NavBarProps { name: ReactNode; links: NavItem[]; serial?: string; status?: ReactNode; cta?: ReactNode; homeHref?: string; label?: string; className?: string; }
export declare function NavBar(props: NavBarProps): JSX.Element;

export interface HeroProps { name: ReactNode; role?: ReactNode; lead?: ReactNode; status?: ReactNode; actions?: ReactNode; specs?: SpecItem[]; portrait?: ReactNode; portraitLabel?: string; serial?: string; coords?: string; className?: string; }
export declare function Hero(props: HeroProps): JSX.Element;

export interface ExperienceEntryProps { role: ReactNode; org: ReactNode; start: string; end?: string; current?: boolean; code?: string; location?: string; points?: ReactNode[]; tags?: string[]; className?: string; }
export declare function ExperienceEntry(props: ExperienceEntryProps): JSX.Element;

export interface ProjectLink { label: string; href?: string; }
export interface ProjectCardProps { title: ReactNode; code?: string; image?: string; imageAlt?: string; kind?: string; status?: string; statusTone?: "signal" | "tint" | "outline"; description?: ReactNode; tags?: string[]; links?: ProjectLink[]; hingeLeft?: string; style?: CSSProperties; className?: string; }
export declare function ProjectCard(props: ProjectCardProps): JSX.Element;

export interface ContactBannerProps { title: ReactNode; text?: ReactNode; code?: string; action?: ReactNode; className?: string; }
export declare function ContactBanner(props: ContactBannerProps): JSX.Element;

export interface FooterProps { links?: ProjectLink[]; left?: ReactNode; right?: ReactNode; rulerLabels?: string[]; className?: string; }
export declare function Footer(props: FooterProps): JSX.Element;

/* --- Ledger/01 kit components (mirrors packages/ui) --- */

export interface CategoryIconProps { icon: string; color: string; size?: "sm" | "md" | "lg"; label?: string; className?: string; }
export declare function CategoryIcon(props: CategoryIconProps): JSX.Element;

export interface ColorDotProps { color: string; className?: string; }
export declare function ColorDot(props: ColorDotProps): JSX.Element;

export interface AmountProps { minor: bigint; currency: string; minorUnits: number; sign?: "auto" | "always" | "never"; code?: boolean; size?: "sm" | "md" | "lg" | "xl"; tone?: "auto" | "plain" | "muted"; className?: string; }
export declare function Amount(props: AmountProps): JSX.Element;

export interface MoneyInputProps { currency: string; value?: string; onChange?: (e: { target: { value: string } }) => void; id?: string; placeholder?: string; disabled?: boolean; className?: string; }
export declare function MoneyInput(props: MoneyInputProps): JSX.Element;

export interface FxState { from: string; to: string; rate: string; ratePrecise: string; order: ("from" | "to" | "rate")[]; }
export interface FxConverterProps { from: { code: string; minorUnits: number }; to: { code: string; minorUnits: number }; value: FxState; onChange: (next: FxState) => void; invalid?: boolean; className?: string; }
export declare function FxConverter(props: FxConverterProps): JSX.Element;

export interface FieldProps { label: ReactNode; htmlFor?: string; error?: ReactNode; help?: ReactNode; className?: string; children: ReactNode; }
export declare function Field(props: FieldProps): JSX.Element;

export interface TextInputProps { id?: string; value?: string; defaultValue?: string; placeholder?: string; type?: string; disabled?: boolean; "aria-invalid"?: boolean; className?: string; onChange?: (e: { target: { value: string } }) => void; }
export declare function TextInput(props: TextInputProps): JSX.Element;

/** react-day-picker props (mode, selected, onSelect, month, onMonthChange, disabled, …). */
export interface CalendarProps { mode?: "single" | "multiple" | "range"; selected?: unknown; onSelect?: (value: unknown) => void; month?: Date; onMonthChange?: (month: Date) => void; weekStartsOn?: 0 | 1 | 2 | 3 | 4 | 5 | 6; showOutsideDays?: boolean; className?: string; }
export declare function Calendar(props: CalendarProps): JSX.Element;

export interface FieldErrorProps { className?: string; children?: ReactNode; }
/** HUD fault line (LED + FAULT microprint + message). Field renders it from its error prop. */
export declare function FieldError(props: FieldErrorProps): JSX.Element | null;

/** Makes a control reject input: stepped servo jitter + one danger ping, then focus. Target: an element or its id; a [data-fault-group] faults every [data-fault-part] inside. */
export declare function fault(target: Element | string | null | undefined, opts?: { focus?: boolean }): HTMLElement[];

export interface ComboboxProps<T> { value: string; onChange: (key: string) => void; options?: T[]; groups?: { heading?: string; options: T[] }[]; getKey: (option: T) => string; getSearchText?: (option: T) => string; renderOption: (option: T) => ReactNode; renderValue?: (option: T) => ReactNode; placeholder?: ReactNode; searchPlaceholder?: string; emptyText?: ReactNode; footer?: ReactNode; id?: string; invalid?: boolean; disabled?: boolean; className?: string; }
/** Searchable single choice. Typing on the trigger opens it with that search; a pick moves focus to the next field. */
export declare function Combobox<T>(props: ComboboxProps<T>): JSX.Element;
export interface MultiComboboxProps<T> { value: string[]; onChange: (keys: string[]) => void; options?: T[]; groups?: { heading?: string; options: T[] }[]; getKey: (option: T) => string; getSearchText?: (option: T) => string; renderOption: (option: T) => ReactNode; allLabel: string; noun: string; searchPlaceholder?: string; emptyText?: ReactNode; id?: string; invalid?: boolean; disabled?: boolean; className?: string; }
/** Searchable multi-choice picker; nothing selected means "all". Stays open while toggling. */
export declare function MultiCombobox<T>(props: MultiComboboxProps<T>): JSX.Element;
export interface BudgetMeterProps { spent: bigint; budget: bigint; elapsed?: number; label: string; size?: "sm" | "md"; className?: string; }
/** Sunken track filled to the share spent (signal, danger when over) with a today tick. */
export declare function BudgetMeter(props: BudgetMeterProps): JSX.Element;

export interface ChartMoney { currency: string; minorUnits: number; }
export interface BalanceChartProps { points: { label: string; tooltipLabel: string; balance: bigint }[]; money: ChartMoney; className?: string; }
/** Single series: the balance line (chart-line) over a signal-tint area, crosshair tooltip. */
export declare function BalanceChart(props: BalanceChartProps): JSX.Element;
export interface FlowChartProps { points: { label: string; tooltipLabel: string; income: bigint; expense: bigint }[]; money: ChartMoney; only?: "income" | "expense"; className?: string; }
/** Income bars above the baseline, expenses below (chart-income / chart-expense), legend unless only one. */
export declare function FlowChart(props: FlowChartProps): JSX.Element;
export interface DonutChartProps { slices: { key: string; label: string; color: string; amount: bigint }[]; money: ChartMoney; centerLabel: string; className?: string; }
/** Shares by category color; top 7 + Other, 2px shell gaps, total in the center. */
export declare function DonutChart(props: DonutChartProps): JSX.Element;

export interface DatePickerProps { value: string; onChange: (value: string) => void; id?: string; placeholder?: string; disabled?: boolean; className?: string; }
export declare function DatePicker(props: DatePickerProps): JSX.Element;
