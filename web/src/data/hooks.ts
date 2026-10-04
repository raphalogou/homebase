// The hooks screens use to read and change data. Nothing outside this folder
// touches the store, IndexedDB or the network.

import { useCallback, useEffect, useMemo, useState, useSyncExternalStore } from "react";
import { todayIn } from "../lib/dates.ts";
import { ulid } from "../lib/ulid.ts";
import { api, NetworkError } from "./api.ts";
import { useData } from "./provider.tsx";
import {
  goalProgress,
  hasRoom,
  isInbox,
  live,
  openGoals,
  plannedOn,
  rankAtEnd,
  reorder,
  suggestions,
} from "./rules.ts";
import type { Goal, LocalDate, Project, Repeat, Task } from "./types.ts";

export { MAX_PER_DAY } from "./rules.ts";

export function useVersion(): number {
  const { store } = useData();
  return useSyncExternalStore(store.subscribe, store.getVersion);
}

function browserZone(): string {
  return Intl.DateTimeFormat().resolvedOptions().timeZone;
}

/**
 * Today in the user's zone (settings.tz), re-checked every minute so the
 * screens turn over at midnight without a reload.
 */
export function useToday(): LocalDate {
  const { store } = useData();
  useVersion();
  const tz = store.me?.tz ?? browserZone();
  const [today, setToday] = useState(() => todayIn(tz));
  useEffect(() => {
    setToday(todayIn(tz));
    const id = setInterval(() => setToday(todayIn(tz)), 60_000);
    return () => clearInterval(id);
  }, [tz]);
  return today;
}

export function useWeekStart(): 0 | 1 {
  const { store } = useData();
  useVersion();
  return store.me?.weekStart ?? 1;
}

export function useTasks(): Task[] {
  const { store } = useData();
  const v = useVersion();
  // biome-ignore lint/correctness/useExhaustiveDependencies: v is the store's change counter
  return useMemo(() => live(store.tables.tasks.values()), [store, v]);
}

export function useGoals(): Map<string, Goal> {
  const { store } = useData();
  const v = useVersion();
  // biome-ignore lint/correctness/useExhaustiveDependencies: v is the store's change counter
  return useMemo(() => new Map(store.tables.goals), [store, v]);
}

export function useProjects(): Map<string, Project> {
  const { store } = useData();
  const v = useVersion();
  // biome-ignore lint/correctness/useExhaustiveDependencies: v is the store's change counter
  return useMemo(() => new Map(store.tables.projects), [store, v]);
}

export function useRepeats(): Map<string, Repeat> {
  const { store } = useData();
  const v = useVersion();
  // biome-ignore lint/correctness/useExhaustiveDependencies: v is the store's change counter
  return useMemo(() => new Map(store.tables.repeats), [store, v]);
}

export function useTask(id: string | null): Task | null {
  const tasks = useTasks();
  return useMemo(() => tasks.find((t) => t.id === id) ?? null, [tasks, id]);
}

export function useOpenGoals(): Goal[] {
  const goals = useGoals();
  return useMemo(() => openGoals(goals.values()), [goals]);
}

export function useGoalProgress(): (goalId: string) => { done: number; total: number } {
  const tasks = useTasks();
  const projects = useProjects();
  return useCallback((goalId: string) => goalProgress(goalId, tasks, projects), [tasks, projects]);
}

/** Today's picks in rank order, done ones included. */
export function usePlanned(day: LocalDate): Task[] {
  const tasks = useTasks();
  return useMemo(() => plannedOn(tasks, day), [tasks, day]);
}

export function useInbox(): Task[] {
  const tasks = useTasks();
  return useMemo(() => tasks.filter(isInbox).sort((a, b) => b.createdAt - a.createdAt), [tasks]);
}

export function useSuggestions() {
  const tasks = useTasks();
  const projects = useProjects();
  const goals = useGoals();
  const today = useToday();
  return useMemo(() => suggestions(tasks, projects, goals, today), [tasks, projects, goals, today]);
}

/** The project or goal a task belongs to, for its "why" line. */
export function useContextOf(): (t: Task) => string | null {
  const projects = useProjects();
  const goals = useGoals();
  return useCallback(
    (t: Task) => {
      if (t.projectId) return projects.get(t.projectId)?.title ?? null;
      if (t.goalId) return goals.get(t.goalId)?.title ?? null;
      return null;
    },
    [projects, goals],
  );
}

export interface NewTask {
  title: string;
  goalId?: string | null;
  projectId?: string | null;
  plannedOn?: LocalDate | null;
  due?: LocalDate | null;
  repeat?: Pick<Repeat, "freq" | "every" | "mode"> | null;
}

/** Why an action could not be done. */
export class ActionError extends Error {}

export function useActions() {
  const { store, engine } = useData();
  const today = useToday();

  return useMemo(() => {
    const tasks = () => live(store.tables.tasks.values());
    const get = (id: string) => {
      const t = store.tables.tasks.get(id);
      if (!t) throw new ActionError("That task no longer exists.");
      return t;
    };
    const update = (id: string, patch: Partial<Task>) =>
      store.put("tasks", { ...get(id), ...patch });

    const plan = async (id: string, day: LocalDate) => {
      const t = get(id);
      if (t.plannedOn === day) return;
      if (!hasRoom(tasks(), day, id)) throw new ActionError("That day already has three tasks.");
      await update(id, { plannedOn: day, planRank: rankAtEnd(tasks(), day) });
    };

    return {
      async addTask(n: NewTask): Promise<string> {
        const title = n.title.trim();
        if (!title) throw new ActionError("Type a task first.");
        if (n.plannedOn && !hasRoom(tasks(), n.plannedOn))
          throw new ActionError("That day already has three tasks.");
        const now = Date.now();
        let repeatId: string | null = null;
        if (n.repeat) {
          repeatId = ulid(now);
          await store.put("repeats", {
            id: repeatId,
            weekdays: null,
            until: null,
            createdAt: now,
            updatedAt: now,
            deletedAt: null,
            rev: 0,
            ...n.repeat,
          });
        }
        const id = ulid(now);
        await store.put("tasks", {
          id,
          projectId: n.projectId ?? null,
          goalId: n.projectId ? null : (n.goalId ?? null),
          title: title.slice(0, 300),
          notes: "",
          status: "open",
          due: n.due ?? null,
          plannedOn: n.plannedOn ?? null,
          planRank: n.plannedOn ? rankAtEnd(tasks(), n.plannedOn) : null,
          slipped: 0,
          repeatId,
          doneAt: null,
          createdAt: now,
          updatedAt: now,
          deletedAt: null,
          rev: 0,
        });
        return id;
      },

      update,

      async toggleDone(id: string) {
        const t = get(id);
        const done = t.status !== "done";
        await update(id, { status: done ? "done" : "open", doneAt: done ? Date.now() : null });
      },

      plan,

      /** Takes a task off its day; the returned function puts it back. */
      async unplan(id: string): Promise<() => Promise<void>> {
        const before = store.snapshot([{ table: "tasks", id }]);
        await update(id, { plannedOn: null, planRank: null });
        return () => store.restore(before);
      },

      /** Makes ids today's open picks, in order. Done picks stay where they are. */
      async setToday(ids: string[]) {
        const current = plannedOn(tasks(), today);
        const keep = new Set(ids);
        for (const t of current) {
          if (t.status === "open" && !keep.has(t.id)) {
            await update(t.id, { plannedOn: null, planRank: null });
          }
        }
        for (const id of ids) {
          await plan(id, today);
        }
      },

      /** Moves a task to position `to` among the tasks planned on its day. */
      async move(id: string, to: number) {
        const t = get(id);
        if (!t.plannedOn) return;
        for (const { id: tid, rank } of reorder(plannedOn(tasks(), t.plannedOn), id, to)) {
          await update(tid, { planRank: rank });
        }
      },

      /** Sets the task's repeat rule; fields left out keep their value (or none). */
      async setRepeat(
        id: string,
        repeat:
          | (Pick<Repeat, "freq" | "every" | "mode"> & Partial<Pick<Repeat, "weekdays" | "until">>)
          | null,
      ) {
        const t = get(id);
        const now = Date.now();
        if (!repeat) {
          // Deleting the rule stops the chain; finished instances keep their link.
          if (t.repeatId) await store.remove("repeats", t.repeatId);
          await update(id, { repeatId: null });
          return;
        }
        const existing = t.repeatId ? store.tables.repeats.get(t.repeatId) : undefined;
        const repeatId = existing?.id ?? ulid(now);
        await store.put("repeats", {
          id: repeatId,
          weekdays: existing?.weekdays ?? null,
          until: existing?.until ?? null,
          createdAt: existing?.createdAt ?? now,
          updatedAt: now,
          deletedAt: null,
          rev: existing?.rev ?? 0,
          ...repeat,
        });
        if (t.repeatId !== repeatId) await update(id, { repeatId });
      },

      /** Deletes a task and its attachments; the returned function undoes it. */
      async remove(id: string): Promise<() => Promise<void>> {
        const atts = live(store.tables.attachments.values()).filter((a) => a.taskId === id);
        const before = store.snapshot([
          { table: "tasks", id },
          ...atts.map((a) => ({ table: "attachments" as const, id: a.id })),
        ]);
        await store.remove("tasks", id);
        return () => store.restore(before);
      },

      /**
       * Turns an Inbox task into a project. Online it uses /api/promote, which
       * does it in one server transaction; offline the same change travels as
       * ordinary ops, which the server also applies in one transaction.
       */
      async promote(id: string, goalId: string | null): Promise<string> {
        const t = get(id);
        if (store.outbox.size === 0) {
          try {
            const res = await api.promote(id, goalId);
            // Show the result at once: a sync already under way would not
            // include it, and the next one brings the server's own rows.
            const at = Date.now();
            await store.anticipate("projects", [res.project]);
            await store.anticipate("tasks", [{ ...t, deletedAt: at }]);
            await store.anticipate(
              "attachments",
              live(store.tables.attachments.values())
                .filter((a) => a.taskId === id)
                .map((a) => ({ ...a, taskId: null, projectId: res.project.id })),
            );
            void engine.run();
            return res.project.id;
          } catch (err) {
            if (!(err instanceof NetworkError)) throw err;
          }
        }
        const now = Date.now();
        const projectId = ulid(now);
        await store.put("projects", {
          id: projectId,
          goalId: goalId ?? t.goalId,
          title: t.title,
          notes: t.notes,
          status: "open",
          due: t.due,
          createdAt: now,
          updatedAt: now,
          deletedAt: null,
          rev: 0,
        });
        for (const a of store.tables.attachments.values()) {
          if (a.taskId === id && a.deletedAt === null) {
            await store.put("attachments", { ...a, taskId: null, projectId });
          }
        }
        await store.remove("tasks", id);
        return projectId;
      },
    };
  }, [store, engine, today]);
}

export function useAuth() {
  const { auth, login, setup, logout, store } = useData();
  useVersion();
  return {
    auth,
    login,
    setup,
    logout,
    // Logging out empties the local copy; a session that ended keeps it, and
    // its outbox, until the next login sends it.
    sessionEnded: auth === "signedOut" && (store.rev > 0 || store.outbox.size > 0),
    lastUsername: store.me?.username ?? "",
  };
}
