// Goals, projects and attachments: reading them and changing them.

import { useEffect, useMemo } from "react";
import { ulid } from "../lib/ulid.ts";
import { normaliseUrl } from "../lib/urls.ts";
import { ApiError, api, NetworkError } from "./api.ts";
import { ActionError, useTasks, useVersion } from "./hooks.ts";
import { useData } from "./provider.tsx";
import { live } from "./rules.ts";
import type { Attachment, Goal, GoalStatus, Project, Task } from "./types.ts";

export type OwnerKind = "goal" | "project" | "task";
export interface Owner {
  kind: OwnerKind;
  id: string;
}

export function useGoal(id: string | undefined): Goal | null {
  const { store } = useData();
  useVersion();
  const g = id ? store.tables.goals.get(id) : undefined;
  return g && g.deletedAt === null ? g : null;
}

export function useProject(id: string | undefined): Project | null {
  const { store } = useData();
  useVersion();
  const p = id ? store.tables.projects.get(id) : undefined;
  return p && p.deletedAt === null ? p : null;
}

/** Live projects; the caller filters by status. */
export function useProjectList(): Project[] {
  const { store } = useData();
  const v = useVersion();
  // biome-ignore lint/correctness/useExhaustiveDependencies: v is the store's change counter
  return useMemo(
    () => live(store.tables.projects.values()).sort((a, b) => a.createdAt - b.createdAt),
    [store, v],
  );
}

export function useAttachments(owner: Owner): Attachment[] {
  const { store } = useData();
  const v = useVersion();
  // biome-ignore lint/correctness/useExhaustiveDependencies: v is the store's change counter
  return useMemo(() => {
    const key =
      owner.kind === "goal" ? "goalId" : owner.kind === "project" ? "projectId" : "taskId";
    return live(store.tables.attachments.values())
      .filter((a) => a[key] === owner.id)
      .sort((a, b) => a.createdAt - b.createdAt);
  }, [store, v, owner.kind, owner.id]);
}

/** Tasks of a project, open ones first in the order they were made. */
export function useProjectTasks(projectId: string): Task[] {
  const tasks = useTasks();
  return useMemo(
    () => sortForList(tasks.filter((t) => t.projectId === projectId)),
    [tasks, projectId],
  );
}

/** Tasks directly under a goal, not through a project. */
export function useGoalTasks(goalId: string): Task[] {
  const tasks = useTasks();
  return useMemo(() => sortForList(tasks.filter((t) => t.goalId === goalId)), [tasks, goalId]);
}

function sortForList(list: Task[]): Task[] {
  const rank = { open: 0, done: 1, dropped: 2 } as const;
  return list
    .filter((t) => t.status !== "dropped")
    .sort((a, b) => rank[a.status] - rank[b.status] || a.createdAt - b.createdAt);
}

/** Done over all non-dropped tasks of one project. */
export function projectProgress(projectId: string, tasks: Task[]): { done: number; total: number } {
  const list = tasks.filter((t) => t.projectId === projectId && t.status !== "dropped");
  return { done: list.filter((t) => t.status === "done").length, total: list.length };
}

/** The first open task of a project, as "Next" under its name. */
export function nextTask(projectId: string, tasks: Task[]): Task | null {
  return (
    tasks
      .filter((t) => t.projectId === projectId && t.status === "open")
      .sort((a, b) => (a.planRank ?? 1e9) - (b.planRank ?? 1e9) || a.createdAt - b.createdAt)[0] ??
    null
  );
}

/** Type and size of an uploaded file, fetched once and remembered. */
export function useFileMeta(sha: string | null): { type: string; size: number } | null | undefined {
  const { store } = useData();
  useVersion();
  useEffect(() => {
    if (!sha || store.fileMeta.has(sha)) return;
    api.fileMeta(sha).then(
      (m) => store.setFileMeta(sha, m),
      () => {},
    );
  }, [sha, store]);
  return sha ? store.fileMeta.get(sha) : undefined;
}

/** What a delete would take with it, for the question before deleting. */
export function useContents() {
  const { store } = useData();
  useVersion();
  return {
    ofProject(id: string) {
      const tasks = live(store.tables.tasks.values()).filter((t) => t.projectId === id);
      const atts = live(store.tables.attachments.values()).filter((a) => a.projectId === id);
      return { projects: 0, tasks: tasks.length, attachments: atts.length };
    },
    ofGoal(id: string) {
      const projects = live(store.tables.projects.values()).filter((p) => p.goalId === id);
      const projectIds = new Set(projects.map((p) => p.id));
      const tasks = live(store.tables.tasks.values()).filter(
        (t) => t.goalId === id || (t.projectId !== null && projectIds.has(t.projectId)),
      );
      const atts = live(store.tables.attachments.values()).filter((a) => a.goalId === id);
      return { projects: projects.length, tasks: tasks.length, attachments: atts.length };
    },
  };
}

function ownerFields(owner: Owner) {
  return {
    goalId: owner.kind === "goal" ? owner.id : null,
    projectId: owner.kind === "project" ? owner.id : null,
    taskId: owner.kind === "task" ? owner.id : null,
  };
}

export function useStructureActions() {
  const { store, engine } = useData();

  return useMemo(() => {
    const now = () => Date.now();
    const getProject = (id: string) => {
      const p = store.tables.projects.get(id);
      if (!p) throw new ActionError("That project no longer exists.");
      return p;
    };
    const getGoal = (id: string) => {
      const g = store.tables.goals.get(id);
      if (!g) throw new ActionError("That goal no longer exists.");
      return g;
    };
    const liveAttachments = () => live(store.tables.attachments.values());

    /** Every row a delete of this goal or project can touch, for Undo. */
    function affected(kind: "goal" | "project", id: string) {
      const tasks = live(store.tables.tasks.values());
      const projectIds = new Set(
        kind === "project"
          ? [id]
          : live(store.tables.projects.values())
              .filter((p) => p.goalId === id)
              .map((p) => p.id),
      );
      const taskIds = tasks
        .filter(
          (t) =>
            (t.projectId !== null && projectIds.has(t.projectId)) ||
            (kind === "goal" && t.goalId === id),
        )
        .map((t) => t.id);
      const taskSet = new Set(taskIds);
      const attIds = liveAttachments()
        .filter(
          (a) =>
            (kind === "goal" && a.goalId === id) ||
            (a.projectId !== null && projectIds.has(a.projectId)) ||
            (a.taskId !== null && taskSet.has(a.taskId)),
        )
        .map((a) => a.id);
      return [
        ...(kind === "goal" ? [{ table: "goals" as const, id }] : []),
        ...[...projectIds].map((p) => ({ table: "projects" as const, id: p })),
        ...taskIds.map((t) => ({ table: "tasks" as const, id: t })),
        ...attIds.map((a) => ({ table: "attachments" as const, id: a })),
      ];
    }

    /** Mirrors the server's cascade (docs/SPEC.md section 3) so the screens update at once. */
    async function anticipateDelete(kind: "goal" | "project", id: string, withContents: boolean) {
      const at = now();
      const gone = <T extends { deletedAt: number | null }>(r: T): T => ({ ...r, deletedAt: at });
      const tasks = live(store.tables.tasks.values());
      const projects = live(store.tables.projects.values());
      const childProjects = kind === "goal" ? projects.filter((p) => p.goalId === id) : [];
      const projectIds = new Set(
        kind === "project" ? [id] : withContents ? childProjects.map((p) => p.id) : [],
      );

      const taskRows: Task[] = [];
      for (const t of tasks) {
        if (t.projectId !== null && projectIds.has(t.projectId)) {
          taskRows.push(withContents ? gone(t) : { ...t, projectId: null });
        } else if (kind === "goal" && t.goalId === id) {
          taskRows.push(withContents ? gone(t) : { ...t, goalId: null });
        }
      }
      const doomedTasks = new Set(taskRows.filter((t) => t.deletedAt !== null).map((t) => t.id));
      const attRows = liveAttachments()
        .filter(
          (a) =>
            (kind === "goal" && a.goalId === id) ||
            (a.projectId !== null && (projectIds.has(a.projectId) || a.projectId === id)) ||
            (a.taskId !== null && doomedTasks.has(a.taskId)),
        )
        .map(gone);
      const projectRows = childProjects.map((p) =>
        withContents ? gone(p) : { ...p, goalId: null },
      );

      await store.anticipate("tasks", taskRows);
      await store.anticipate("attachments", attRows);
      await store.anticipate("projects", projectRows);
    }

    return {
      async addGoal(title: string): Promise<string> {
        const clean = title.trim();
        if (!clean) throw new ActionError("Type a goal first.");
        const at = now();
        const id = ulid(at);
        const last = Math.max(0, ...live(store.tables.goals.values()).map((g) => g.sortKey));
        await store.put("goals", {
          id,
          title: clean.slice(0, 300),
          notes: "",
          status: "open",
          targetDate: null,
          sortKey: last + 1,
          createdAt: at,
          updatedAt: at,
          deletedAt: null,
          rev: 0,
        });
        return id;
      },

      updateGoal: (id: string, patch: Partial<Goal>) =>
        store.put("goals", { ...getGoal(id), ...patch }),

      async addProject(title: string, goalId: string | null): Promise<string> {
        const clean = title.trim();
        if (!clean) throw new ActionError("Type a project name first.");
        const at = now();
        const id = ulid(at);
        await store.put("projects", {
          id,
          goalId,
          title: clean.slice(0, 300),
          notes: "",
          status: "open",
          due: null,
          createdAt: at,
          updatedAt: at,
          deletedAt: null,
          rev: 0,
        });
        return id;
      },

      updateProject: (id: string, patch: Partial<Project>) =>
        store.put("projects", { ...getProject(id), ...patch }),

      setStatus(kind: "goal" | "project", id: string, status: GoalStatus) {
        return kind === "goal"
          ? store.put("goals", { ...getGoal(id), status })
          : store.put("projects", { ...getProject(id), status });
      },

      /** Deletes a project; the returned function puts everything back. */
      async deleteProject(id: string, withContents: boolean): Promise<() => Promise<void>> {
        const before = store.snapshot(affected("project", id));
        await anticipateDelete("project", id, withContents);
        await store.remove("projects", id, withContents ? "delete" : "detach");
        return () => store.restore(before);
      },

      /** Deletes a goal; the returned function puts everything back. */
      async deleteGoal(id: string, withContents: boolean): Promise<() => Promise<void>> {
        const before = store.snapshot(affected("goal", id));
        await anticipateDelete("goal", id, withContents);
        await store.remove("goals", id, withContents ? "delete" : "detach");
        return () => store.restore(before);
      },

      async addLink(owner: Owner, url: string, name: string) {
        const href = normaliseUrl(url);
        if (!href) throw new ActionError("Enter a web address, such as example.com.");
        const at = now();
        await store.put("attachments", {
          id: ulid(at),
          ...ownerFields(owner),
          kind: "link",
          name: name.trim().slice(0, 300),
          url: href,
          body: null,
          fileSha: null,
          createdAt: at,
          updatedAt: at,
          deletedAt: null,
          rev: 0,
        });
      },

      async addNote(owner: Owner, body: string) {
        if (!body.trim()) throw new ActionError("Write the note first.");
        const at = now();
        await store.put("attachments", {
          id: ulid(at),
          ...ownerFields(owner),
          kind: "note",
          name: "",
          url: null,
          body: body.slice(0, 20000),
          fileSha: null,
          createdAt: at,
          updatedAt: at,
          deletedAt: null,
          rev: 0,
        });
      },

      async updateNote(id: string, body: string) {
        const a = store.tables.attachments.get(id);
        if (!a || a.body === body) return;
        await store.put("attachments", { ...a, body: body.slice(0, 20000) });
      },

      async removeAttachment(id: string): Promise<() => Promise<void>> {
        const before = store.snapshot([{ table: "attachments", id }]);
        await store.remove("attachments", id);
        return () => store.restore(before);
      },

      /**
       * Uploads need the server: the bytes go straight to it. The new
       * attachment row comes back and joins the local copy.
       */
      async upload(owner: Owner, file: File): Promise<void> {
        if (file.size > 25 * 1024 * 1024) throw new ActionError("Files can be at most 25 MB.");
        if (file.size === 0) throw new ActionError("That file is empty.");
        try {
          // An owner made offline must reach the server before its file can.
          if (store.outbox.size > 0) await engine.run();
          const att = await api.upload(owner.kind, owner.id, file);
          await store.apply({
            goals: [],
            projects: [],
            tasks: [],
            repeats: [],
            attachments: [att],
            reminders: [],
          });
        } catch (err) {
          if (err instanceof NetworkError)
            throw new ActionError("Uploading needs a connection. Try again when online.");
          if (err instanceof ApiError) {
            const msg =
              err.status === 404
                ? "Sync first: the server does not have this yet. Try again in a moment."
                : err.message;
            throw new ActionError(msg);
          }
          throw err;
        }
      },
    };
  }, [store, engine]);
}
