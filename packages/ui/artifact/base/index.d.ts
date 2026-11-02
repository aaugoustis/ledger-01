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
