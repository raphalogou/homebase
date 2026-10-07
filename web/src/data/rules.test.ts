import assert from "node:assert/strict";
import { test } from "node:test";
import {
  doneByDay,
  goalProgress,
  hasRoom,
  isInbox,
  plannedOn,
  rankAtEnd,
  reorder,
  suggestions,
} from "./rules.ts";
import type { Goal, Project, Task } from "./types.ts";

let seq = 0;
function task(over: Partial<Task> = {}): Task {
  seq++;
  return {
    id: `T${String(seq).padStart(3, "0")}`,
    projectId: null,
    goalId: null,
    title: "Task",
    notes: "",
    status: "open",
    due: null,
    plannedOn: null,
    planRank: null,
    slipped: 0,
    repeatId: null,
    doneAt: null,
    createdAt: seq,
    updatedAt: seq,
    deletedAt: null,
    rev: seq,
    ...over,
  };
}

function goal(id: string, over: Partial<Goal> = {}): Goal {
  return {
    id,
    title: "Goal",
    notes: "",
    status: "open",
    targetDate: null,
    sortKey: 0,
    createdAt: 0,
    updatedAt: 0,
    deletedAt: null,
    rev: 0,
    ...over,
  };
}

function project(id: string, goalId: string | null, over: Partial<Project> = {}): Project {
  return {
    id,
    goalId,
    title: "Project",
    notes: "",
    status: "open",
    due: null,
    createdAt: 0,
    updatedAt: 0,
    deletedAt: null,
    rev: 0,
    ...over,
  };
}

const today = "2026-03-04";

test("isInbox", () => {
  const cases: [Partial<Task>, boolean][] = [
    [{}, true],
    [{ goalId: "G" }, false],
    [{ projectId: "P" }, false],
    [{ due: today }, false],
    [{ plannedOn: today }, false],
    [{ status: "done" }, false],
    [{ deletedAt: 1 }, false],
  ];
  for (const [over, want] of cases) {
    assert.equal(isInbox(task(over)), want, JSON.stringify(over));
  }
});

test("plannedOn counts done tasks, not dropped or deleted ones, in rank order", () => {
  const a = task({ plannedOn: today, planRank: 2 });
  const b = task({ plannedOn: today, planRank: 1, status: "done" });
  const c = task({ plannedOn: today, planRank: 0, status: "dropped" });
  const d = task({ plannedOn: today, planRank: 3, deletedAt: 5 });
  const e = task({ plannedOn: "2026-03-05", planRank: 1 });
  assert.deepEqual(
    plannedOn([a, b, c, d, e], today).map((t) => t.id),
    [b.id, a.id],
  );
});

test("hasRoom stops at three and ignores the task being moved", () => {
  const three = [1, 2, 3].map((r) => task({ plannedOn: today, planRank: r }));
  assert.equal(hasRoom(three.slice(0, 2), today), true);
  assert.equal(hasRoom(three, today), false);
  assert.equal(hasRoom(three, today, three[0]?.id), true);
  assert.equal(hasRoom(three, "2026-03-05"), true);
});

test("rankAtEnd", () => {
  assert.equal(rankAtEnd([], today), 1);
  assert.equal(rankAtEnd([task({ plannedOn: today, planRank: 4.5 })], today), 5.5);
});

test("reorder takes the midpoint of the new neighbours", () => {
  const list = [1, 2, 3].map((r) => task({ plannedOn: today, planRank: r }));
  const [a, b, c] = list;
  assert.ok(a && b && c);
  assert.deepEqual(reorder(list, c.id, 0), [{ id: c.id, rank: 0 }]);
  assert.deepEqual(reorder(list, a.id, 2), [{ id: a.id, rank: 4 }]);
  assert.deepEqual(reorder(list, a.id, 1), [{ id: a.id, rank: 2.5 }]);
  assert.deepEqual(reorder(list, a.id, 0), []);
  assert.deepEqual(reorder(list, a.id, 7), []);
});

test("reorder renumbers when the gap is too small", () => {
  const list = [1, 1 + 1e-7, 2].map((r) => task({ plannedOn: today, planRank: r }));
  const [a, b, c] = list;
  assert.ok(a && b && c);
  assert.deepEqual(reorder(list, c.id, 1), [
    { id: a.id, rank: 1 },
    { id: c.id, rank: 2 },
    { id: b.id, rank: 3 },
  ]);
});

test("suggestions groups each task once and skips today's picks", () => {
  const goals = new Map([
    ["G", goal("G")],
    ["GP", goal("GP", { status: "paused" })],
  ]);
  const projects = new Map([["P", project("P", "G")]]);
  const dueSoon = task({ due: "2026-03-08", slipped: 3 });
  const overdue = task({ due: "2026-03-01" });
  const later = task({ due: "2026-04-01" });
  const moved = task({ slipped: 2 });
  const inProject = task({ projectId: "P" });
  const underPausedGoal = task({ goalId: "GP" });
  const planned = task({ projectId: "P", plannedOn: today });
  const done = task({ due: today, status: "done" });
  const inbox = task();

  const s = suggestions(
    [dueSoon, overdue, later, moved, inProject, underPausedGoal, planned, done, inbox],
    projects,
    goals,
    today,
  );
  assert.deepEqual(
    s.dueSoon.map((t) => t.id),
    [overdue.id, dueSoon.id],
  );
  assert.deepEqual(
    s.moved.map((t) => t.id),
    [moved.id],
  );
  assert.deepEqual(
    s.inProgress.map((t) => t.id),
    [inProject.id],
  );
});

test("goalProgress counts tasks through projects and directly, without dropped ones", () => {
  const projects = new Map([
    ["P", project("P", "G")],
    ["Q", project("Q", "G", { deletedAt: 1 })],
  ]);
  const tasks = [
    task({ goalId: "G", status: "done" }),
    task({ goalId: "G" }),
    task({ projectId: "P", status: "done" }),
    task({ projectId: "P", status: "dropped" }),
    task({ projectId: "Q" }),
    task({ goalId: "OTHER", status: "done" }),
  ];
  assert.deepEqual(goalProgress("G", tasks, projects), { done: 2, total: 3 });
});

test("the Done filter groups by the local day a task was finished, newest first", () => {
  // 2026-03-04 00:30 in Paris is still 3 March in UTC.
  const at = (iso: string) => Date.parse(iso);
  const tasks = [
    task({ id: "a", status: "done", doneAt: at("2026-03-03T23:30:00Z") }),
    task({ id: "b", status: "done", doneAt: at("2026-03-04T09:00:00Z") }),
    task({ id: "c", status: "done", doneAt: at("2026-03-02T10:00:00Z") }),
    task({ id: "old", status: "done", doneAt: at("2026-01-10T10:00:00Z") }),
    task({ id: "open", status: "open" }),
    task({ id: "gone", status: "done", doneAt: at("2026-03-04T08:00:00Z"), deletedAt: 1 }),
  ];
  const { groups, older } = doneByDay(tasks, "Europe/Paris", today, 30);
  assert.deepEqual(
    groups.map((g) => [g.day, g.tasks.map((t) => t.id)]),
    [
      ["2026-03-04", ["b", "a"]],
      ["2026-03-02", ["c"]],
    ],
  );
  assert.equal(older, true);
  assert.equal(doneByDay(tasks, "Europe/Paris", today, 60).older, false);
});
