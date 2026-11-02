/* @ds-bundle: {"format":4,"namespace":"Module01","components":[{"name":"Button"},{"name":"Tag"},{"name":"StatusLed"},{"name":"HudLabel"},{"name":"TickRuler"},{"name":"Crosshair"},{"name":"Grill"},{"name":"SectionHeader"},{"name":"Panel"},{"name":"GlassPanel"},{"name":"SpecReadout"},{"name":"Terminal"},{"name":"NavBar"},{"name":"Hero"},{"name":"ExperienceEntry"},{"name":"ProjectCard"},{"name":"ContactBanner"},{"name":"Footer"}]} */
(function () {
  var R = window.React, h = R.createElement;
  function cx() { return Array.prototype.filter.call(arguments, Boolean).join(" "); }
  function omit(o, keys) { var r = {}; for (var k in o) if (keys.indexOf(k) < 0) r[k] = o[k]; return r; }

  function Button(p) {
    var v = p.variant || "primary";
    var rest = omit(p, ["variant", "href", "className", "children", "arrow"]);
    rest.className = cx("m1-btn", "m1-btn-" + v, p.className);
    var kids = [p.children, p.arrow ? h("span", { key: "a", className: "m1-btn-arrow", "aria-hidden": "true" }, "→") : null];
    if (p.href) { rest.href = p.href; return h("a", rest, kids); }
    if (!rest.type) rest.type = "button";
    return h("button", rest, kids);
  }
  function Tag(p) {
    return h("span", { className: cx("m1-tag", p.tone && "m1-tag-" + p.tone, p.className) },
      p.led ? h("span", { className: "m1-led-dot", style: { width: 6, height: 6 } }) : null, p.children);
  }
  function StatusLed(p) {
    return h("span", { className: cx("m1-led", p.tone && "is-" + p.tone, p.className) },
      h("span", { className: "m1-led-dot", "aria-hidden": "true" }), h("span", { className: "m1-hud" }, p.children));
  }
  function HudLabel(p) {
    return h("div", { className: cx("m1-hudlabel m1-hud", p.className) },
      p.code ? h("span", { className: "m1-hudlabel-code" }, p.code) : null,
      h("span", null, p.children),
      p.rule === false ? null : h("span", { className: "m1-hudlabel-rule", "aria-hidden": "true" }),
      p.end ? h("span", { className: "m1-micro" }, p.end) : null);
  }
  function TickRuler(p) {
    var labels = p.labels || null;
    return h("div", { className: cx("m1-rulerwrap", p.className), "aria-hidden": "true" },
      h("div", { className: "m1-ruler" }),
      labels ? h("div", { className: "m1-ruler-labels m1-micro" }, labels.map(function (l, i) { return h("span", { key: i }, l); })) : null);
  }
  function Crosshair(p) {
    return h("svg", { className: cx("m1-cross", p.tone && "is-" + p.tone, p.className), viewBox: "0 0 12 12", style: p.style, "aria-hidden": "true" },
      h("path", { d: "M6 0v12M0 6h12", stroke: "currentColor", strokeWidth: 1.25, fill: "none" }));
  }
  function Grill(p) {
    var n = p.count || 6, arr = [];
    for (var i = 0; i < n; i++) arr.push(h("span", { key: i, className: "m1-grill-slot", style: p.slotWidth ? { width: p.slotWidth } : null }));
    return h("div", { className: cx("m1-grill", p.tone && "is-" + p.tone, p.className), style: p.gap ? { "--g": p.gap } : null, "aria-hidden": "true" }, arr);
  }
  function SectionHeader(p) {
    return h("header", { className: cx("m1-section", p.className) },
      h(HudLabel, { code: p.code, end: p.index }, p.label || p.title),
      h("div", { className: "m1-section-top" },
        h(p.as || "h2", { className: "m1-h m1-h-lg" }, p.title),
        p.aside ? h("div", null, p.aside) : null),
      p.ruler === false ? null : h(TickRuler, null));
  }
  function Panel(p) {
    var notch = p.notch === undefined ? "md" : p.notch;
    return h("div", { className: cx("m1-panel m1-lift", p.serial && "has-serial", p.className), style: p.style },
      h("div", { className: cx("m1-panel-body", notch && "m1-notch", notch && notch !== "md" && "m1-notch-" + notch, p.recess && "is-recess") },
        p.serial ? h("span", { className: "m1-panel-serial m1-micro" }, p.serial) : null,
        p.children,
        p.cross === false ? null : h(Crosshair, { className: "m1-panel-cross" })));
  }
  function GlassPanel(p) { return h("div", { className: cx("m1-glass", p.className), style: p.style }, p.children); }
  function SpecReadout(p) {
    return h("dl", { className: cx("m1-spec", p.className) }, (p.items || []).map(function (it, i) {
      return h("div", { key: i }, h("dt", { className: "m1-hud" }, it.label), h("dd", null, it.value));
    }));
  }
  function Terminal(p) {
    return h("div", { className: cx("m1-visor", p.className) },
      p.title ? h("div", { className: "m1-micro", style: { marginBottom: 8 } }, p.title) : null,
      (p.lines || []).map(function (l, i) { return h("div", { key: i }, h("span", { className: "m1-prompt" }, "$ "), l); }));
  }
  function NavBar(p) {
    return h("nav", { className: cx("m1-nav", p.className), "aria-label": p.label || "Main" },
      h("a", { className: "m1-nav-id", href: p.homeHref || "#" },
        h("span", { className: "m1-h m1-h-sm" }, p.name),
        p.serial ? h("span", { className: "m1-micro" }, p.serial) : null),
      h("div", { className: "m1-nav-links" }, (p.links || []).map(function (l, i) {
        return h("a", { key: i, className: "m1-nav-link", href: l.href || "#", "aria-current": l.active ? "page" : undefined },
          h("span", { className: "m1-micro" }, String(i + 1).padStart(2, "0")), l.label);
      })),
      p.status ? h(StatusLed, null, p.status) : null,
      p.cta || null);
  }
  function Hero(p) {
    return h("section", { className: cx("m1-hero m1-panel m1-lift", p.className) },
      h("span", { className: "m1-hero-tab", "aria-hidden": "true" }),
      h("div", { className: "m1-panel-body m1-notch m1-notch-lg" },
        p.serial ? h("span", { className: "m1-panel-serial m1-micro", style: { left: "var(--space-16)", top: "24px" } }, p.serial) : null,
        h("div", { className: "m1-hero-main" },
          p.status ? h(StatusLed, null, p.status) : null,
          h("h1", { className: "m1-h m1-h-xl" }, p.name),
          p.role ? h("p", { className: "m1-hud", style: { margin: 0, color: "var(--signal-ink)" } }, p.role) : null,
          p.lead ? h("p", { className: "m1-hero-lead" }, p.lead) : null,
          p.actions ? h("div", { className: "m1-hero-actions" }, p.actions) : null,
          p.specs ? h(SpecReadout, { items: p.specs }) : null),
        h("div", { className: "m1-hero-side" },
          h("div", { className: "m1-hero-port" },
            p.portrait || h(Crosshair, { style: { width: 20, height: 20, color: "rgba(28,30,33,0.45)" } }),
            h("span", { className: "m1-hero-port-ring", "aria-hidden": "true" }),
            p.portraitLabel ? h("span", { className: "m1-hero-port-tag m1-micro", style: { color: "var(--ink)" } }, p.portraitLabel) : null),
          h(Grill, { count: 4, slotWidth: 28, gap: "8px" }),
          p.coords ? h("span", { className: "m1-micro" }, p.coords) : null),
        h(Crosshair, { className: "m1-panel-cross", tone: "signal" })));
  }
  function ExperienceEntry(p) {
    return h("article", { className: cx("m1-exp", p.className) },
      h("div", { className: "m1-exp-when" },
        h("span", { className: "m1-hudnum" }, p.start + " — " + (p.end || "Now")),
        p.code ? h("span", { className: "m1-micro" }, p.code) : null,
        p.location ? h("span", { className: "m1-micro", style: { color: "var(--ink-soft)" } }, p.location) : null),
      h("div", { className: "m1-exp-track", "aria-hidden": "true" }, h("span", { className: cx("m1-exp-node", !p.current && "is-past") })),
      h("div", { className: "m1-exp-body" },
        h("h3", { className: "m1-h m1-h-sm" }, p.role),
        h("p", { className: "m1-exp-org" }, p.org),
        p.points && p.points.length ? h("ul", { className: "m1-exp-list" }, p.points.map(function (t, i) { return h("li", { key: i }, t); })) : null,
        p.tags ? h("div", { className: "m1-exp-tags" }, p.tags.map(function (t, i) { return h(Tag, { key: i }, t); })) : null));
  }
  function ProjectCard(p) {
    var st = p.status ? h(Tag, { tone: p.statusTone || (p.status.toLowerCase() === "live" ? null : "tint"), led: p.status.toLowerCase() === "live" }, p.status) : null;
    return h("article", { className: cx("m1-proj m1-panel m1-lift", p.className), style: p.style },
      h("span", { className: "m1-hinge", style: { left: p.hingeLeft || "28px" }, "aria-hidden": "true" }),
      h("div", { className: "m1-panel-body m1-notch" },
        h("div", { className: "m1-proj-media" },
          p.image ? h("img", { src: p.image, alt: p.imageAlt || "" }) : null,
          h(Crosshair, { style: { top: 12, left: 12 } }), h(Crosshair, { style: { bottom: 12, right: 12 } }),
          h("span", { className: "m1-micro", style: { position: "absolute", top: 12, right: 30, color: "var(--ink)" } }, p.code)),
        h("div", { className: "m1-proj-meta" }, h("span", { className: "m1-hud", style: { color: "var(--ink-soft)" } }, p.kind || ""), st),
        h("div", { className: "m1-proj-text" },
          h("h3", { className: "m1-h m1-h-sm" }, p.title),
          p.description ? h("p", { className: "m1-proj-desc" }, p.description) : null),
        p.tags ? h("div", { className: "m1-proj-tags" }, p.tags.map(function (t, i) { return h(Tag, { key: i, tone: "outline" }, t); })) : null,
        p.links ? h("div", { className: "m1-proj-links" }, p.links.map(function (l, i) { return h(Button, { key: i, variant: "ghost", href: l.href || "#" }, l.label); })) : null));
  }
  function ContactBanner(p) {
    return h("section", { className: cx("m1-contact m1-panel m1-lift", p.className) },
      h("div", { className: "m1-panel-body m1-notch m1-notch-lg" },
        h("div", null,
          p.code ? h("div", { className: "m1-micro", style: { marginBottom: 16 } }, p.code) : null,
          h("h2", { className: "m1-h m1-h-md" }, p.title),
          p.text ? h("p", { className: "m1-contact-sub" }, p.text) : null),
        h("div", { className: "m1-contact-side" },
          h(Grill, { count: 6, tone: "on-signal", slotWidth: 30, gap: "8px" }),
          p.action || null)));
  }
  function Footer(p) {
    return h("footer", { className: cx("m1-footer", p.className) },
      h(TickRuler, { labels: p.rulerLabels }),
      h("div", { className: "m1-footer-row" },
        h("span", { className: "m1-micro is-read" }, p.left),
        h("div", { className: "m1-footer-links" }, (p.links || []).map(function (l, i) { return h(Button, { key: i, variant: "ghost", href: l.href || "#" }, l.label); })),
        h("span", { className: "m1-micro is-read" }, p.right)));
  }

  window.Module01 = Object.assign(window.Module01 || {}, {
    Button: Button, Tag: Tag, StatusLed: StatusLed, HudLabel: HudLabel, TickRuler: TickRuler, Crosshair: Crosshair, Grill: Grill,
    SectionHeader: SectionHeader, Panel: Panel, GlassPanel: GlassPanel, SpecReadout: SpecReadout, Terminal: Terminal, NavBar: NavBar,
    Hero: Hero, ExperienceEntry: ExperienceEntry, ProjectCard: ProjectCard, ContactBanner: ContactBanner, Footer: Footer
  });
})();
