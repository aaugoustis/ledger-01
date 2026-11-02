// Fault feedback: makes a control visibly reject input (servo jitter + danger ping, see the
// "Fault feedback" section of styles/kit.css) and moves focus to it.
//
// Targets: an element or an element id. A target inside a [data-fault-part] (e.g. the input in
// a MoneyInput) faults that whole part; a [data-fault-group] (e.g. the FxConverter) faults every
// part inside it.

/** Longest fault animation (ms); the attribute is removed after it. */
export const FAULT_MS = 900;

const timers = new WeakMap<HTMLElement, number>();

function restart(part: HTMLElement) {
  const t = timers.get(part);
  if (t !== undefined) window.clearTimeout(t);
  part.removeAttribute("data-fault");
  // Force a reflow so the animation starts again even if it was still running.
  void part.offsetWidth;
  part.setAttribute("data-fault", "");
  timers.set(
    part,
    window.setTimeout(() => {
      part.removeAttribute("data-fault");
      timers.delete(part);
    }, FAULT_MS),
  );
}

/** Faults a control. Returns the parts that were animated. */
export function fault(target: Element | string | null | undefined, opts: { focus?: boolean } = {}) {
  if (typeof document === "undefined" || !target) return [];
  const el = typeof target === "string" ? document.getElementById(target) : target;
  if (!(el instanceof HTMLElement)) return [];

  const parts = el.matches("[data-fault-group]")
    ? Array.from(el.querySelectorAll<HTMLElement>("[data-fault-part]"))
    : [el.closest<HTMLElement>("[data-fault-part]") ?? el];
  for (const p of parts) restart(p);

  if (opts.focus !== false) {
    const focusable = el.matches("input, button, select, textarea, [tabindex]")
      ? el
      : el.querySelector<HTMLElement>("input, button, select, textarea, [tabindex]");
    focusable?.focus({ preventScroll: false });
  }
  return parts;
}
