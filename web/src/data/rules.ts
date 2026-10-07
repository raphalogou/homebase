// Client copies of the rules in docs/SPEC.md section 5. The server decides;
// these let the screens answer instantly and offline.

import { addDays, daysBetween, todayIn } from "../lib/dates.ts";
import type { Goal, LocalDate, Project, Task } from "./types.ts";

export const MAX_PER_DAY = 3;
/** Ranks closer than this are renumbered (docs/SPEC.md section 1). */
export const MIN_RANK_GAP = 1e-6;

type Row = { deletedAt: number | null };

export function live<T extends Row>(rows: Iterable<T>): T[] {
  return [...rows].filter((r) => r.deletedAt === null);
}

/** An open, live task with no project, no goal, no due and no planned day. */
export function isInbox(t: Task): boolean {
  return (
    t.deletedAt === null &&
    t.status === "open" &&
    t.projectId === null &&
    t.goalId === null &&
    t.due === null &&
    t.plannedOn === null
  );
}

function byRank(a: Task, b: Task): number {
  const ra = a.planRank ?? Number.POSITIVE_INFINITY;
  const rb = b.planRank ?? Number.POSITIVE_INFINITY;
  return ra - rb || a.createdAt - b.createdAt || a.id.localeCompare(b.id);
}

/** Tasks that take one of day's slots, in rank order. Done ones count. */
export function plannedOn(tasks: Iterable<Task>, day: LocalDate): Task[] {
  return live(tasks)
    .filter((t) => t.plannedOn === day && t.status !== "dropped")
    .sort(byRank);
}

export function hasRoom(tasks: Iterable<Task>, day: LocalDate, taskId?: string): boolean {
  return plannedOn(tasks, day).filter((t) => t.id !== taskId).length < MAX_PER_DAY;
}

/** The rank for a task added at the end of day's list. */
export function rankAtEnd(tasks: Iterable<Task>, day: LocalDate): number {
  const list = plannedOn(tasks, day);
  const last = list[list.length - 1]?.planRank;
  return last == null ? 1 : last + 1;
}

/**
 * The new ranks after moving task `id` to position `to` in `list` (already
 * in rank order). Usually one task changes, taking the midpoint of its new
 * neighbours; when they are too close, the whole day is renumbered.
 */
export function reorder(list: Task[], id: string, to: number): { id: string; rank: number }[] {
  const from = list.findIndex((t) => t.id === id);
  if (from < 0 || to < 0 || to >= list.length || from === to) return [];
  const rest = list.filter((t) => t.id !== id);
  const prev = rest[to - 1]?.planRank ?? null;
  const next = rest[to]?.planRank ?? null;

  let rank: number | null;
  if (prev === null && next === null) rank = 1;
  else if (prev === null) rank = (next ?? 0) - 1;
  else if (next === null) rank = prev + 1;
  else rank = next - prev < MIN_RANK_GAP * 2 ? null : (prev + next) / 2;

  if (rank !== null) return [{ id, rank }];
  const moved = list[from];
  if (!moved) return [];
  const order = [...rest.slice(0, to), moved, ...rest.slice(to)];
  return order.map((t, i) => ({ id: t.id, rank: i + 1 }));
}

export interface Suggestions {
  dueSoon: Task[];
  inProgress: Task[];
  moved: Task[];
}

/** How far ahead "Due soon" looks. */
export const DUE_SOON_DAYS = 7;

/**
 * Candidates for today's three, in the groups of DESIGN.md. Each task shows
 * once: due soon first, then moved a few times, then in progress (a task with
 * an open project or goal). Tasks already planned for today are left out.
 */
export function suggestions(
  tasks: Iterable<Task>,
  projects: Map<string, Project>,
  goals: Map<string, Goal>,
  today: LocalDate,
): Suggestions {
  const out: Suggestions = { dueSoon: [], inProgress: [], moved: [] };
  const horizon = addDays(today, DUE_SOON_DAYS);
  const open = live(tasks).filter((t) => t.status === "open" && t.plannedOn !== today);

  for (const t of open) {
    if (t.due !== null && t.due <= horizon) {
      out.dueSoon.push(t);
    } else if (t.slipped >= 2) {
      out.moved.push(t);
    } else if (hasOpenParent(t, projects, goals)) {
      out.inProgress.push(t);
    }
  }
  out.dueSoon.sort((a, b) => (a.due ?? "").localeCompare(b.due ?? "") || a.createdAt - b.createdAt);
  out.moved.sort((a, b) => b.slipped - a.slipped || a.createdAt - b.createdAt);
  out.inProgress.sort((a, b) => b.updatedAt - a.updatedAt);
  return out;
}

function hasOpenParent(t: Task, projects: Map<string, Project>, goals: Map<string, Goal>): boolean {
  if (t.projectId !== null) {
    const p = projects.get(t.projectId);
    return p !== undefined && p.deletedAt === null && p.status === "open";
  }
  if (t.goalId !== null) {
    const g = goals.get(t.goalId);
    return g !== undefined && g.deletedAt === null && g.status === "open";
  }
  return false;
}

/** The goal a task serves, directly or through its project. */
export function goalOf(t: Task, projects: Map<string, Project>): string | null {
  if (t.goalId !== null) return t.goalId;
  if (t.projectId !== null) return projects.get(t.projectId)?.goalId ?? null;
  return null;
}

/** Done tasks over all non-dropped tasks, from the goal's projects and its own tasks. */
export function goalProgress(
  goalId: string,
  tasks: Iterable<Task>,
  projects: Map<string, Project>,
): { done: number; total: number } {
  let done = 0;
  let total = 0;
  for (const t of live(tasks)) {
    if (t.status === "dropped" || goalOf(t, projects) !== goalId) continue;
    if (t.projectId !== null && projects.get(t.projectId)?.deletedAt != null) continue;
    total++;
    if (t.status === "done") done++;
  }
  return { done, total };
}

/** Open goals in their display order. */
export function openGoals(goals: Iterable<Goal>): Goal[] {
  return live(goals)
    .filter((g) => g.status === "open")
    .sort((a, b) => a.sortKey - b.sortKey || a.createdAt - b.createdAt);
}

/** Tasks for the Plan list: open or done-today, grouped later by day. */
export function dayOf(t: Task): LocalDate | null {
  return t.plannedOn ?? t.due;
}

export function isOverdue(t: Task, today: LocalDate): boolean {
  return t.status === "open" && t.due !== null && daysBetween(today, t.due) < 0;
}

export interface DoneDay {
  day: LocalDate;
  tasks: Task[];
}

/**
 * Plan's Done filter: tasks finished in the last `days` days up to today,
 * by the local day they were finished in tz, newest first. `older` says
 * whether any were finished before that, for "Show older".
 */
export function doneByDay(
  tasks: Task[],
  tz: string,
  today: LocalDate,
  days: number,
): { groups: DoneDay[]; older: boolean } {
  const from = addDays(today, -(days - 1));
  const when = (t: Task) => t.doneAt ?? t.updatedAt;
  const byDay = new Map<LocalDate, Task[]>();
  let older = false;
  for (const t of tasks) {
    if (t.status !== "done" || t.deletedAt !== null) continue;
    const day = todayIn(tz, when(t));
    if (day < from) {
      older = true;
      continue;
    }
    byDay.set(day, [...(byDay.get(day) ?? []), t]);
  }
  const groups = [...byDay]
    .sort(([a], [b]) => b.localeCompare(a))
    .map(([day, list]) => ({ day, tasks: list.sort((a, b) => when(b) - when(a)) }));
  return { groups, older };
}
