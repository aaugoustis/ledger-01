import { describe, expect, test } from "bun:test";
import { startingWallet } from "./last-wallet";

const wallet = (id: string, opts: { hidden?: boolean; archived?: boolean } = {}) => ({
  id,
  hidden: opts.hidden ?? false,
  archived: opts.archived ?? false,
});

describe("startingWallet", () => {
  const card = wallet("card");
  const savings = wallet("savings", { hidden: true });
  const old = wallet("old", { archived: true });

  test("the wallet last added to", () => {
    expect(startingWallet([card, savings], "savings")).toBe(savings);
  });

  test("the first wallet the dashboard shows when there's no last one", () => {
    expect(startingWallet([savings, card], null)).toBe(card);
    expect(startingWallet([savings, card], "deleted")).toBe(card);
  });

  test("never an archived wallet", () => {
    expect(startingWallet([old, card], "old")).toBe(card);
    expect(startingWallet([old], null)).toBeUndefined();
  });

  test("a hidden wallet when it's the only one left", () => {
    expect(startingWallet([old, savings], null)).toBe(savings);
  });

  test("nothing without wallets", () => {
    expect(startingWallet([], "card")).toBeUndefined();
  });
});
