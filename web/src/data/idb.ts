// IndexedDB mirror of the synced tables, the outbox, and a few settings.

import { type DBSchema, type IDBPDatabase, openDB } from "idb";
import type { Attachment, Goal, Me, Op, Project, Reminder, Repeat, Task } from "./types.ts";

export interface OutboxEntry {
  /** "table/id": one entry per row, so later changes replace earlier ones. */
  key: string;
  seq: number;
  op: Op;
}

export interface MetaValues {
  rev: number;
  me: Me;
  /** True once a login succeeded, so the app opens offline next time. */
  signedIn: boolean;
  /** True once a first full sync finished, so screens stop showing skeletons. */
  loaded: boolean;
}

interface Schema extends DBSchema {
  goals: { key: string; value: Goal };
  projects: { key: string; value: Project };
  tasks: { key: string; value: Task };
  repeats: { key: string; value: Repeat };
  attachments: { key: string; value: Attachment };
  reminders: { key: number; value: Reminder };
  outbox: { key: string; value: OutboxEntry };
  meta: { key: string; value: unknown };
}

export type DB = IDBPDatabase<Schema>;

export const ROW_STORES = ["goals", "projects", "tasks", "repeats", "attachments"] as const;
export type RowStore = (typeof ROW_STORES)[number];

export function openDatabase(name = "homebase"): Promise<DB> {
  return openDB<Schema>(name, 1, {
    upgrade(db) {
      for (const store of ROW_STORES) {
        db.createObjectStore(store, { keyPath: "id" });
      }
      db.createObjectStore("reminders", { keyPath: "slot" });
      db.createObjectStore("outbox", { keyPath: "key" });
      db.createObjectStore("meta");
    },
  });
}

export async function getMeta<K extends keyof MetaValues>(
  db: DB,
  key: K,
): Promise<MetaValues[K] | undefined> {
  return (await db.get("meta", key)) as MetaValues[K] | undefined;
}

export async function setMeta<K extends keyof MetaValues>(
  db: DB,
  key: K,
  value: MetaValues[K],
): Promise<void> {
  await db.put("meta", value, key);
}

/** Removes everything, for logging out on a shared device. */
export async function clearAll(db: DB): Promise<void> {
  const names = [...ROW_STORES, "reminders", "outbox", "meta"] as const;
  const tx = db.transaction(names, "readwrite");
  await Promise.all([...names.map((n) => tx.objectStore(n).clear()), tx.done]);
}
