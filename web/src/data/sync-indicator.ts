// The rules of the offline and sync indicator, kept free of React so they
// can be tested on their own. sync-status.ts feeds them from the engine.

import type { SyncState } from "./sync.ts";

export type SyncIndicator =
  | { kind: "none" }
  | { kind: "syncing" }
  | { kind: "offline"; waiting: number }
  | { kind: "error"; retryIn: number }
  | { kind: "recovered" };

export interface IndicatorInput {
  state: SyncState;
  /** The sync under way has taken over a second. */
  slow: boolean;
  /** The last trouble seen and not yet cleared by a good sync. */
  trouble: "offline" | "error" | null;
  /** A good sync just ended trouble, within the last three seconds. */
  recovered: boolean;
  waiting: number;
  retryIn: number;
}

/** Pure, so the rules can be tested without timers. */
export function indicatorFor(i: IndicatorInput): SyncIndicator {
  const trouble = i.state === "offline" || i.state === "error" ? i.state : i.trouble;
  // A retry under way keeps the trouble showing until it has been slow for
  // a second, so quick failing retries do not flicker.
  if (i.state === "syncing" && i.slow && trouble) return { kind: "syncing" };
  if (trouble === "offline") return { kind: "offline", waiting: i.waiting };
  if (trouble === "error") return { kind: "error", retryIn: i.retryIn };
  if (i.recovered) return { kind: "recovered" };
  if (i.state === "syncing" && i.slow) return { kind: "syncing" };
  return { kind: "none" };
}
