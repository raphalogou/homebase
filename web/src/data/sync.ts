// When and how the local copy talks to the server (docs/SPEC.md section 3).

import { type Api, ApiError, NetworkError } from "./api.ts";
import type { LocalStore } from "./store.ts";
import type { SyncTable } from "./types.ts";

export type SyncState = "idle" | "syncing" | "offline" | "error";

const DEBOUNCE_MS = 800;
const INTERVAL_MS = 60_000;
const MAX_BACKOFF_MS = 60_000;
const PUSH_BATCH = 500;

export class SyncEngine {
  state: SyncState = "idle";
  private running: Promise<void> | null = null;
  private again = false;
  private debounce: ReturnType<typeof setTimeout> | undefined;
  private retry: ReturnType<typeof setTimeout> | undefined;
  private backoff = 2000;
  private stopFns: (() => void)[] = [];

  private readonly store: LocalStore;
  private readonly api: Api;
  private readonly onSignedOut: () => void;

  constructor(store: LocalStore, api: Api, onSignedOut: () => void) {
    this.store = store;
    this.api = api;
    this.onSignedOut = onSignedOut;
    store.setOnWrite(() => this.soon());
  }

  /** Starts the triggers: focus, coming online, visibility, every minute. */
  start(): void {
    const now = () => void this.run();
    const visible = () => {
      if (document.visibilityState === "visible") void this.run();
    };
    window.addEventListener("focus", now);
    window.addEventListener("online", now);
    document.addEventListener("visibilitychange", visible);
    const timer = setInterval(() => {
      if (document.visibilityState === "visible") void this.run();
    }, INTERVAL_MS);
    // The service worker forwards push messages that carry the sync hint.
    const message = (e: MessageEvent) => {
      if ((e.data as { type?: string } | null)?.type === "sync") void this.run();
    };
    navigator.serviceWorker?.addEventListener("message", message);

    this.stopFns.push(
      () => window.removeEventListener("focus", now),
      () => window.removeEventListener("online", now),
      () => document.removeEventListener("visibilitychange", visible),
      () => clearInterval(timer),
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
    this.state = "syncing";
    try {
      await this.push();
      await this.pull();
      this.backoff = 2000;
      this.state = "idle";
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        this.state = "idle";
        this.onSignedOut();
        return;
      }
      this.state = err instanceof NetworkError ? "offline" : "error";
      // Retry with backoff; the online event also wakes us sooner.
      this.retry = setTimeout(() => void this.run(), this.backoff);
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

  private async pull(): Promise<void> {
    for (;;) {
      const page = await this.api.pull(this.store.rev);
      await this.store.apply(page, page.rev);
      if (!page.more) return;
    }
  }
}
