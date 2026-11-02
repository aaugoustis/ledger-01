// Form flow: keyboard focus moves through a form the way a person fills it in.
//
// - Dialogs and sheets focus their first field when they open (not the close button or tabs).
// - Single-choice controls (Select, Combobox, DatePicker) move focus to the next field after a
//   choice is committed, instead of returning it to their own trigger.
// - Tab follows DOM order, so fields are laid out in the order they are filled.
//
// A "field" is any enabled text input, textarea, native select, combobox trigger, or element
// marked [data-flow-field]. Checkboxes, tabs and plain buttons are not fields.

const FIELD_SELECTOR = [
  "input:not([type=hidden]):not([type=checkbox]):not([type=radio])",
  "textarea",
  "select",
  "[role=combobox]",
  "[data-flow-field]",
]
  .map((s) => `${s}:not([disabled]):not([aria-disabled=true])`)
  .join(",");

function visible(el: HTMLElement) {
  return el.getClientRects().length > 0 && getComputedStyle(el).visibility !== "hidden";
}

/** Fields inside a container, in DOM (tab) order. */
export function flowFields(root: ParentNode): HTMLElement[] {
  return Array.from(root.querySelectorAll<HTMLElement>(FIELD_SELECTOR)).filter(
    (el) => visible(el) && !el.closest("[data-flow-skip]"),
  );
}

/**
 * Focuses the first field inside a container. If that field is an empty picker (marked
 * [data-flow-autoopen]), it also opens it once the container's entrance animation has finished,
 * so the list doesn't anchor to a trigger that is still sliding in. Returns whether a field was
 * found.
 */
export function focusFirstField(root: ParentNode | null | undefined): boolean {
  const first = root ? flowFields(root)[0] : undefined;
  if (!first) return false;
  first.focus();
  if (first.hasAttribute("data-flow-autoopen")) openWhenSettled(first);
  return true;
}

function openWhenSettled(trigger: HTMLElement) {
  const scope = trigger.closest<HTMLElement>("[role=dialog]");
  const running = scope?.getAnimations?.() ?? [];
  void Promise.allSettled(running.map((a) => a.finished)).then(() => {
    // Only if the person hasn't moved on in the meantime.
    if (trigger.isConnected && document.activeElement === trigger) trigger.click();
  });
}

/**
 * Moves focus to the field after `from` within its form (or dialog). Falls back to the form's
 * submit button, then to `from` itself. Returns the element that got focus.
 */
export function focusNextField(from: HTMLElement | null | undefined): HTMLElement | null {
  if (!from) return null;
  const scope =
    from.closest<HTMLElement>("form, [data-flow-scope], [role=dialog]") ?? from.ownerDocument.body;
  const fields = flowFields(scope);
  const index = fields.findIndex((f) => f === from || f.contains(from) || from.contains(f));
  const next =
    (index >= 0 ? fields[index + 1] : undefined) ??
    scope.querySelector<HTMLElement>("button[type=submit]:not([disabled])") ??
    from;
  next.focus();
  return next;
}
