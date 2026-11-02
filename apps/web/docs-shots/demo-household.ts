// A made-up household for the docs screenshots: four months of everyday spending, a salary, rent,
// transfers into a peso account and savings, budgets and two recurring rules. Every name, merchant
// and amount is invented. Dates are relative to `today`, so the screenshots never look stale.

type Id = string;
type Category = { id: Id; kind: "expense" | "income"; name: string };
// The parts of a backup this file reads and writes (backend/internal/ledger/backup.go).
export type BackupDoc = {
  household: { name: string; base_currency: string };
  categories: Category[];
  [key: string]: unknown;
};

const iso = (d: Date) => d.toISOString().slice(0, 10);
const addDays = (d: Date, n: number) => new Date(d.getTime() + n * 86_400_000);
const monthDay = (today: Date, monthsBack: number, day: number) =>
  new Date(Date.UTC(today.getUTCFullYear(), today.getUTCMonth() - monthsBack, day));

// Deterministic, so a re-run produces the same pictures.
function rng(seed: number) {
  let s = seed;
  return () => {
    s = (s * 1664525 + 1013904223) % 4294967296;
    return s / 4294967296;
  };
}

export function demoBackup(base: BackupDoc, todayStr: string): BackupDoc {
  const today = new Date(`${todayStr}T00:00:00Z`);
  const rand = rng(7);
  const between = (lo: number, hi: number) => Math.round((lo + rand() * (hi - lo)) * 100);
  const pick = <T>(xs: T[]) => xs[Math.floor(rand() * xs.length)];
  const uuid = () => crypto.randomUUID();
  const createdAt = (d: string) => `${d}T12:00:00Z`;

  const cat = (kind: Category["kind"], name: string) => {
    const c = base.categories.find((x) => x.kind === kind && x.name === name);
    if (!c) throw new Error(`no ${kind} category "${name}" in the base backup`);
    return c.id;
  };

  const wallets = [
    { id: uuid(), name: "Everyday card", currency: "USD", initial_balance_minor: 245000 },
    { id: uuid(), name: "Savings", currency: "USD", initial_balance_minor: 1200000 },
    { id: uuid(), name: "Peso account", currency: "UYU", initial_balance_minor: 1800000 },
    { id: uuid(), name: "Cash", currency: "USD", initial_balance_minor: 18000 },
  ].map((w, position) => ({ ...w, hidden: false, position }));
  const [card, savings, pesos, cash] = wallets.map((w) => w.id);

  const labels = [
    { id: uuid(), name: "Vacation", color: "cat-sky" },
    { id: uuid(), name: "Shared", color: "cat-violet" },
  ];
  const [vacation, shared] = labels.map((l) => l.id);

  const transactions: Record<string, unknown>[] = [];
  const transfers: Record<string, unknown>[] = [];
  const recurringRules: Record<string, unknown>[] = [];

  const tx = (
    wallet: Id,
    kind: "expense" | "income",
    category: Id,
    amount: number,
    date: Date,
    note: string,
    extra: Record<string, unknown> = {},
  ) => {
    const d = iso(date);
    transactions.push({
      id: uuid(),
      wallet_id: wallet,
      kind,
      category_id: category,
      amount_minor: amount,
      occurred_on: d,
      note,
      created_at: createdAt(d),
      ...extra,
    });
  };

  const transfer = (from: Id, to: Id, sent: number, received: number, rate: string, date: Date) => {
    const id = uuid();
    const d = iso(date);
    transfers.push({
      id,
      from_wallet_id: from,
      to_wallet_id: to,
      from_amount_minor: sent,
      to_amount_minor: received,
      rate,
      occurred_on: d,
      note: "",
      created_at: createdAt(d),
    });
    for (const [wallet, kind, amount] of [
      [from, "transfer_out", sent],
      [to, "transfer_in", received],
    ] as const) {
      transactions.push({
        id: uuid(),
        wallet_id: wallet,
        kind,
        transfer_id: id,
        amount_minor: amount,
        occurred_on: d,
        note: "",
        created_at: createdAt(d),
      });
    }
  };

  // Recurring rules, with every occurrence from four months back to three months ahead.
  const monthly = (wallet: Id, category: Id, amount: number, day: number, note: string) => {
    const id = uuid();
    const start = monthDay(today, 4, day);
    let last = start;
    for (let m = 4; m >= -3; m--) {
      const d = monthDay(today, m, day);
      last = d;
      tx(wallet, "expense", category, amount, d, note, {
        recurring_rule_id: id,
        occurrence_date: iso(d),
      });
    }
    recurringRules.push({
      id,
      kind: "expense",
      wallet_id: wallet,
      category_id: category,
      amount_minor: amount,
      note,
      frequency: "monthly",
      start_date: iso(start),
      generated_through: iso(last),
    });
  };
  monthly(card, cat("expense", "Home"), 140000, 3, "Rent · Elm Court");
  monthly(card, cat("expense", "Entertainment"), 1299, 12, "Streamflix");

  for (let m = 4; m >= 0; m--) {
    const first = monthDay(today, m, 1);
    if (first > today) continue;
    tx(card, "income", cat("income", "Salary"), 485000, first, "Salary · Brightline Studio");
    const at = (day: number) => monthDay(today, m, day);
    if (at(8) <= today)
      tx(
        card,
        "expense",
        cat("expense", "Utilities"),
        between(70, 95),
        at(8),
        "Brightwater Energy",
      );
    if (at(10) <= today)
      tx(card, "expense", cat("expense", "Utilities"), 4900, at(10), "Swiftnet Internet");
    if (at(5) <= today) transfer(card, savings, 50000, 50000, "1", at(5));
    if (at(6) <= today) transfer(card, pesos, 30000, 1215000, "40.5", at(6));
    if (m === 2)
      tx(card, "income", cat("income", "Business"), 62000, at(18), "Logo design · Harbor Bakery");
  }

  for (let d = 120; d >= 0; d--) {
    const day = addDays(today, -d);
    const weekday = day.getUTCDay();
    if (weekday === 6 || weekday === 3)
      tx(
        card,
        "expense",
        cat("expense", "Groceries"),
        between(38, 140),
        day,
        pick(["Greenleaf Market", "Corner Pantry"]),
        {
          label_ids: [shared],
        },
      );
    if (rand() < 0.4)
      tx(
        card,
        "expense",
        cat("expense", "Food & Drink"),
        between(6, 48),
        day,
        pick(["Blue Door Café", "Luna Pizza", "Noodle Bar 88", "Morning Bakehouse"]),
      );
    if (weekday === 1)
      tx(card, "expense", cat("expense", "Transport"), between(15, 30), day, "City Transit");
    if (weekday === 0)
      tx(
        pesos,
        "expense",
        cat("expense", "Groceries"),
        between(900, 3400),
        day,
        "Feria del barrio",
        {
          label_ids: [shared],
        },
      );
    if (rand() < 0.06)
      tx(
        card,
        "expense",
        cat("expense", "Shopping"),
        between(25, 160),
        day,
        pick(["Fieldnote Books", "Atlas Outdoor", "Pixel Hardware"]),
      );
    if (rand() < 0.08)
      tx(cash, "expense", cat("expense", "Food & Drink"), between(3, 12), day, "Coffee cart");
  }
  tx(
    card,
    "expense",
    cat("expense", "Healthcare"),
    6000,
    addDays(today, -40),
    "Dr. Ana Pérez · check-up",
  );
  tx(
    card,
    "expense",
    cat("expense", "Travel"),
    48900,
    addDays(today, -75),
    "Skyway Air · round trip",
    { label_ids: [vacation] },
  );
  tx(
    card,
    "expense",
    cat("expense", "Travel"),
    31200,
    addDays(today, -62),
    "Seaside Inn · 3 nights",
    { label_ids: [vacation] },
  );

  const budgets = [
    {
      name: "Eating out",
      amount_minor: 40000,
      categories: [cat("expense", "Food & Drink")],
      wallets: [card, cash],
    },
    {
      name: "Groceries",
      amount_minor: 70000,
      categories: [cat("expense", "Groceries")],
      wallets: [card],
    },
    {
      name: "Getting around",
      amount_minor: 15000,
      categories: [cat("expense", "Transport")],
      wallets: [card],
    },
  ].map((b) => ({
    id: uuid(),
    name: b.name,
    amount_minor: b.amount_minor,
    currency: "USD",
    repeat: "monthly",
    start_date: iso(monthDay(today, 4, 1)),
    wallet_ids: b.wallets,
    category_ids: b.categories,
  }));

  return {
    ...base,
    household: { name: "Maple Street", base_currency: "USD" },
    wallets,
    hidden_categories: [],
    labels,
    transfers,
    recurring_rules: recurringRules,
    transactions,
    budgets,
    import_profiles: [],
    import_rules: [],
    import_refs: [],
  };
}
