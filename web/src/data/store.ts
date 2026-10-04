// The local copy of the user's data. Reads are synchronous from memory;
// every write goes to memory at once and to IndexedDB, together with its
// outbox entry, in one transaction.

import {
  clearAll,
  type DB,
  getMeta,
  type OutboxEntry,
  ROW_STORES,
  type RowStore,
  setMeta,
} from "./idb.ts";
import type { Changes, Me, Op, Reminder, RowOf, SyncTable } from "./types.ts";

type Tables = { [K in SyncTable]: Map<string, RowOf[K]> };
type SnapshotRow = { [K in SyncTable]: { table: K; row: RowOf[K] } }[SyncTable];
export type Snapshot = SnapshotRow[];

export class LocalStore {
  readonly tables: Tables = {
    goals: new Map(),
    projects: new Map(),
    tasks: new Map(),
    repeats: new Map(),
    attachments: new Map(),
  };
  reminders = new Map<number, Reminder>();
  outbox = new Map<string, OutboxEntry>();
  rev = 0;
  me: Me | null = null;
  signedIn = false;
  /** A first full sync has finished on this device since the last login. */
  loaded = false;

  /** Type and size of uploaded files, learned from the server, by SHA. */
  fileMeta = new Map<string, { type: string; size: number } | null>();

  /** Bumped on every change; hooks re-read when it moves. */
  version = 0;
  private seq = 0;
  private listeners = new Set<() => void>();
  private onWrite: () => void = () => {};

  private readonly db: DB;

  constructor(db: DB) {
    this.db = db;
  }

  async load(): Promise<void> {
    for (const name of ROW_STORES) {
      const rows = await this.db.getAll(name);
      const table = this.tables[name] as Map<string, { id: string }>;
      for (const r of rows) table.set(r.id, r);
    }
    for (const r of await this.db.getAll("reminders")) this.reminders.set(r.slot, r);
    for (const e of await this.db.getAll("outbox")) {
      this.outbox.set(e.key, e);
      this.seq = Math.max(this.seq, e.seq);
    }
    this.rev = (await getMeta(this.db, "rev")) ?? 0;
    this.me = (await getMeta(this.db, "me")) ?? null;
    this.signedIn = (await getMeta(this.db, "signedIn")) ?? false;
    // Devices from before this flag existed had their data already.
    this.loaded = (await getMeta(this.db, "loaded")) ?? this.rev > 0;
    this.changed();
  }

  subscribe = (fn: () => void): (() => void) => {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  };

  getVersion = (): number => this.version;

  /** Called after each local write, to schedule a sync. */
  setOnWrite(fn: () => void): void {
    this.onWrite = fn;
  }

  private changed(): void {
    this.version++;
    for (const fn of this.listeners) fn();
  }

  /**
   * Writes a row locally and queues it for the server. updatedAt always
   * moves forward, so the server never sees this as an older edit.
   */
  async put<K extends SyncTable>(table: K, row: RowOf[K]): Promise<void> {
    const prev = this.tables[table].get(row.id);
    const updatedAt = Math.max(Date.now(), (prev?.updatedAt ?? 0) + 1);
    const next = { ...row, updatedAt } as RowOf[K];
    this.tables[table].set(next.id, next);
    await this.persist(table, next, { op: "upsert", table, id: next.id, row: next, updatedAt });
  }

  /** Tombstones a row locally and queues the delete. */
  async remove(table: SyncTable, id: string, cascade?: "delete" | "detach"): Promise<void> {
    const prev = this.tables[table].get(id);
    if (!prev || prev.deletedAt !== null) return;
    const updatedAt = Math.max(Date.now(), prev.updatedAt + 1);
    const next = { ...prev, updatedAt, deletedAt: updatedAt };
    (this.tables[table] as Map<string, typeof next>).set(id, next);
    const op: Op = { op: "delete", table, id, updatedAt };
    if (cascade) op.cascade = cascade;
    await this.persist(table, next, op);
  }

  private async persist(table: RowStore, row: { id: string }, op: Op): Promise<void> {
    const entry: OutboxEntry = { key: `${table}/${row.id}`, seq: ++this.seq, op };
    this.outbox.set(entry.key, entry);
    this.changed();
    const tx = this.db.transaction([table, "outbox"], "readwrite");
    await Promise.all([
      (tx.objectStore(table) as unknown as { put(v: unknown): Promise<unknown> }).put(row),
      tx.objectStore("outbox").put(entry),
      tx.done,
    ]);
    this.onWrite();
  }

  /**
   * Shows a change the server will make anyway, such as a cascade, without
   * queueing it. The server's own rows replace it on the next sync.
   */
  async anticipate<K extends SyncTable>(table: K, rows: RowOf[K][]): Promise<void> {
    if (rows.length === 0) return;
    const tx = this.db.transaction(table, "readwrite");
    const store = tx.store as unknown as { put(v: unknown): Promise<unknown> };
    for (const r of rows) this.tables[table].set(r.id, r);
    await Promise.all([...rows.map((r) => store.put(r)), tx.done]);
    this.changed();
  }

  setFileMeta(sha: string, meta: { type: string; size: number } | null): void {
    this.fileMeta.set(sha, meta);
    this.changed();
  }

  /** Copies of rows as they are now, to put back on Undo. */
  snapshot(rows: { table: SyncTable; id: string }[]): Snapshot {
    const out: Snapshot = [];
    for (const { table, id } of rows) {
      const row = this.tables[table].get(id);
      if (row) out.push({ table, row: { ...row } } as SnapshotRow);
    }
    return out;
  }

  /**
   * Writes a snapshot back as new edits, parents before children. Being
   * later than the change it undoes, it wins on every device.
   */
  async restore(snap: Snapshot): Promise<void> {
    const order: SyncTable[] = ["goals", "projects", "repeats", "tasks", "attachments"];
    for (const table of order) {
      for (const s of snap) {
        if (s.table === table) await this.put(table, s.row as RowOf[typeof table]);
      }
    }
  }

  /** Outbox entries in the order they were made. */
  pending(): OutboxEntry[] {
    return [...this.outbox.values()].sort((a, b) => a.seq - b.seq);
  }

  /** Drops entries the server has answered, unless the row changed again since. */
  async acknowledge(sent: OutboxEntry[]): Promise<void> {
    const done = sent.filter((e) => this.outbox.get(e.key)?.seq === e.seq);
    for (const e of done) this.outbox.delete(e.key);
    const tx = this.db.transaction("outbox", "readwrite");
    await Promise.all([...done.map((e) => tx.store.delete(e.key)), tx.done]);
  }

  /**
   * Takes rows from the server. A row with a change still waiting in the
   * outbox keeps the local version; the server's answer to that change will
   * bring the final one.
   */
  async apply(changes: Changes, rev?: number): Promise<void> {
    const tx = this.db.transaction(
      [...ROW_STORES, "reminders" as const, "meta" as const],
      "readwrite",
    );
    const writes: Promise<unknown>[] = [];
    for (const name of ROW_STORES) {
      const table = this.tables[name] as Map<string, { id: string }>;
      for (const row of changes[name] as { id: string }[]) {
        if (this.outbox.has(`${name}/${row.id}`)) continue;
        table.set(row.id, row);
        writes.push(
          (tx.objectStore(name) as unknown as { put(v: unknown): Promise<unknown> }).put(row),
        );
      }
    }
    for (const r of changes.reminders) {
      this.reminders.set(r.slot, r);
      writes.push(tx.objectStore("reminders").put(r));
    }
    if (rev !== undefined) {
      this.rev = rev;
      writes.push(tx.objectStore("meta").put(rev, "rev"));
    }
    await Promise.all([...writes, tx.done]);
    this.changed();
  }

  /** Forgets a row the server never accepted, such as a fourth task on a full day. */
  async forget(table: SyncTable, id: string): Promise<void> {
    if (this.outbox.has(`${table}/${id}`)) return;
    this.tables[table].delete(id);
    await this.db.delete(table, id);
    this.changed();
  }

  async setMe(me: Me): Promise<void> {
    this.me = me;
    await setMeta(this.db, "me", me);
    this.changed();
  }

  async setLoaded(): Promise<void> {
    if (this.loaded) return;
    this.loaded = true;
    await setMeta(this.db, "loaded", true);
    this.changed();
  }

  async setSignedIn(v: boolean): Promise<void> {
    this.signedIn = v;
    await setMeta(this.db, "signedIn", v);
    this.changed();
  }

  /** Empties memory and IndexedDB, on logout. */
  async reset(): Promise<void> {
    await clearAll(this.db);
    for (const t of Object.values(this.tables)) t.clear();
    this.reminders.clear();
    this.outbox.clear();
    this.rev = 0;
    this.me = null;
    this.signedIn = false;
    this.loaded = false;
    this.changed();
  }
}
