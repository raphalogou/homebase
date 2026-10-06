// When and how the local copy talks to the server (docs/SPEC.md section 3).

import { type Api, ApiError, NetworkError } from "./api.ts";
import type { LocalStore } from "./store.ts";
import type { SyncTable } from "./types.ts";

export type SyncState = "idle" | "syncing" | "offline" | "error";

const DEBOUNCE_MS = 800;
const MAX_BACKOFF_MS = 60_000;
const PUSH_BATCH = 500;
/** Focus, the tab showing and the timer skip a sync this soon after the last one. */
export const QUIET_MS = 15_000;
export const POLL_MIN_MS = 60_000;
export const POLL_MAX_MS = 10 * 60_000;

/**
 * The wait before the next timed check: a minute after anything changed,
 * then twice as long after each quiet check, up to ten minutes. Every
 * request wakes the phone's radio, so a planner left open costs little.
 */
export function nextPoll(prev: number, changed: boolean): number {
  return changed ? POLL_MIN_MS : Math.min(prev * 2, POLL_MAX_MS);
}

export class SyncEngine {
  state: SyncState = "idle";
  /** When the next retry after a failure is due, in ms since the epoch; 0 when none is. */
  retryAt = 0;
  /** True once a sync has completed since the app opened. */
  loaded = false;
  private stateListeners = new Set<() => void>();
  private running: Promise<void> | null = null;
  private again = false;
  private debounce: ReturnType<typeof setTimeout> | undefined;
  private retry: ReturnType<typeof setTimeout> | undefined;
  private backoff = 2000;
  private stopFns: (() => void)[] = [];
  private poll: ReturnType<typeof setTimeout> | undefined;
  private pollDelay = POLL_MIN_MS;
  private lastOk = 0;

  private readonly store: LocalStore;
  private readonly api: Api;
  private readonly onSignedOut: () => void;

  constructor(store: LocalStore, api: Api, onSignedOut: () => void) {
    this.store = store;
    this.api = api;
    this.onSignedOut = onSignedOut;
    store.setOnWrite(() => this.soon());
  }

  /** Listens for changes of state, retryAt or loaded. */
  onState = (fn: () => void): (() => void) => {
    this.stateListeners.add(fn);
    return () => this.stateListeners.delete(fn);
  };

  private setState(state: SyncState, retryAt = 0): void {
    if (state === this.state && retryAt === this.retryAt) return;
    this.state = state;
    this.retryAt = retryAt;
    for (const fn of this.stateListeners) fn();
  }

  /**
   * Starts the triggers. Coming online and a push message sync at once;
   * focus and the tab showing sync unless one just ran (they often fire
   * together); a timer checks while the tab shows, less often while
   * nothing changes (nextPoll).
   */
  start(): void {
    const now = () => void this.run();
    const passive = () => {
      // Focus and the tab showing often fire together: one sync serves both.
      if (
        document.visibilityState === "visible" &&
        !this.running &&
        Date.now() - this.lastOk >= QUIET_MS
      ) {
        void this.run();
      }
    };
    // The browser knows at once; a request would only fail after a timeout.
    const offline = () => this.setState("offline", this.retryAt);
    window.addEventListener("focus", passive);
    window.addEventListener("online", now);
    window.addEventListener("offline", offline);
    document.addEventListener("visibilitychange", passive);
    // The service worker forwards push messages that carry the sync hint.
    const message = (e: MessageEvent) => {
      if ((e.data as { type?: string } | null)?.type === "sync") void this.run();
    };
    navigator.serviceWorker?.addEventListener("message", message);

    this.stopFns.push(
      () => window.removeEventListener("focus", passive),
      () => window.removeEventListener("online", now),
      () => window.removeEventListener("offline", offline),
      () => document.removeEventListener("visibilitychange", passive),
      () => clearTimeout(this.poll),
      () => navigator.serviceWorker?.removeEventListener("message", message),
    );
    void this.run();
  }

  stop(): void {
    for (const fn of this.stopFns) fn();
    this.stopFns = [];
    clearTimeout(this.debounce);
    clearTimeout(this.retry);
  }

  /** Syncs shortly after a local change, so quick edits travel together. */
  soon(): void {
    this.pollDelay = POLL_MIN_MS;
    clearTimeout(this.debounce);
    this.debounce = setTimeout(() => void this.run(), DEBOUNCE_MS);
  }

  /** Runs one sync, or queues one more if a sync is under way. */
  run(): Promise<void> {
    if (this.running) {
      this.again = true;
      return this.running;
    }
    this.running = this.cycle().finally(() => {
      this.running = null;
      if (this.again) {
        this.again = false;
        void this.run();
      }
    });
    return this.running;
  }

  private async cycle(): Promise<void> {
    if (!this.store.signedIn) return;
    clearTimeout(this.retry);
    this.setState("syncing");
    try {
      const pushed = this.store.pending().length > 0;
      await this.push();
      const pulled = await this.pull();
      this.backoff = 2000;
      this.lastOk = Date.now();
      this.schedulePoll(pushed || pulled);
      if (!this.loaded) {
        this.loaded = true;
        await this.store.setLoaded();
      }
      this.setState("idle");
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        this.setState("idle");
        this.onSignedOut();
        return;
      }
      // Retry with backoff; the online event also wakes us sooner.
      this.retry = setTimeout(() => void this.run(), this.backoff);
      this.setState(err instanceof NetworkError ? "offline" : "error", Date.now() + this.backoff);
      this.backoff = Math.min(this.backoff * 2, MAX_BACKOFF_MS);
    }
  }

  private async push(): Promise<void> {
    for (;;) {
      const batch = this.store.pending().slice(0, PUSH_BATCH);
      if (batch.length === 0) return;
      const res = await this.api.push(
        this.store.rev,
        batch.map((e) => e.op),
      );
      await this.store.acknowledge(batch);

      // A rejected new row has no server copy in changes: drop it locally.
      const returned = new Set<string>();
      for (const [table, rows] of Object.entries(res.changes)) {
        for (const r of rows as { id?: string }[]) if (r.id) returned.add(`${table}/${r.id}`);
      }
      await this.store.apply(res.changes, res.rev);
      for (const rej of res.rejected) {
        const op = batch.find((e) => e.op.id === rej.id)?.op;
        if (op && !returned.has(`${op.table}/${op.id}`)) {
          await this.store.forget(op.table as SyncTable, op.id);
        }
      }
    }
  }

  /** Pulls every page; true when any of them brought something. */
  private async pull(): Promise<boolean> {
    let brought = false;
    for (;;) {
      const page = await this.api.pull(this.store.rev);
      brought ||= page.rev !== this.store.rev;
      await this.store.apply(page, page.rev);
      if (!page.more) return brought;
    }
  }

  private schedulePoll(changed: boolean): void {
    clearTimeout(this.poll);
    if (this.stopFns.length === 0) return; // stopped
    this.pollDelay = nextPoll(this.pollDelay, changed);
    this.poll = setTimeout(() => {
      // A hidden tab waits; showing it again syncs through the passive trigger.
      if (document.visibilityState === "visible") void this.run();
    }, this.pollDelay);
  }
}
