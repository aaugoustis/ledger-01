import type { Wallet } from "@ledger-01/api";

const LAST_WALLET = "ledger.last-wallet";

type WalletChoice = Pick<Wallet, "id" | "hidden" | "archived">;

/**
 * The wallet a new transaction starts in when no wallet page is open: the one last added to, else
 * the first wallet the dashboard shows, else any open one. Archived wallets never.
 */
export function startingWallet<W extends WalletChoice>(
  wallets: W[],
  lastId: string | null,
): W | undefined {
  const open = wallets.filter((w) => !w.archived);
  return open.find((w) => w.id === lastId) ?? open.find((w) => !w.hidden) ?? open[0];
}

export function lastWalletId(): string | null {
  try {
    return window.localStorage.getItem(LAST_WALLET);
  } catch {
    return null;
  }
}

export function rememberWallet(id: string) {
  try {
    window.localStorage.setItem(LAST_WALLET, id);
  } catch {
    // Private mode or blocked storage: new transactions start in the first wallet instead.
  }
}
