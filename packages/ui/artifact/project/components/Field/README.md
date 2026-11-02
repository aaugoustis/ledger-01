# Field

A form row: label, one control, then an error or help line. `TextInput` is the plain text well that goes in it; `FieldLabel` is the label on its own (FxConverter uses it).

Provide `label` and `children` (the control), and `htmlFor` with the control's `id`. Optional `error` (turns the label `danger` with a fault LED; the message is announced to screen readers, not shown) or `help` (in `ink-soft`).

- Labels use `body-sm` in `ink-soft`, sentence case, above the control with a 6px gap.
- `TextInput` matches MoneyInput: `shell-recess` well, `shadow-press`, `radius-md`, `control-height`, `body-md` text.
- Put "(optional)" in the label of optional fields instead of marking required ones.
- Forms open in a side sheet on desktop and a bottom sheet on phones.

## Fault feedback

When a submit is rejected, the offending control reacts like a part refusing input. Call `fault(idOrElement)`, then set the Field's `error`.

- **Servo jitter**: 320ms, ±3px horizontally, stepped (it jumps between positions, no easing), so it reads mechanical rather than wobbly.
- **Ping**: one 2px `danger` ring leaves the part's outline and fades over 700ms, like a radar ping. No blur, no glow.
- **Fault label**: the label turns `danger` and gains a 6px `danger` LED with a `danger-tint` halo that blinks three times and stays lit. No line under the control; the toast carries the message and screen readers hear it.
- Focus moves to the control. A MoneyInput faults as one part; an FxConverter faults all three wells.
- Editing the control clears its error. The toast still reports the problem too.
- Reduced motion: no jitter or ping, just a short static `danger` outline.
- Server rejections that name a field (`rate: amounts don't match…`) fault that field the same way.
