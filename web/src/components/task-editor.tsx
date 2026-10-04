import { useEffect, useId, useState } from "react";
import {
  ActionError,
  useActions,
  useGoals,
  useOpenGoals,
  usePlanned,
  useProjects,
  useRepeats,
  useTask,
  useToday,
} from "@/data/hooks";
import { live } from "@/data/rules";
import type { Repeat, Task, TaskStatus } from "@/data/types";
import { useOpenTask } from "@/lib/open-task";
import { useIsDesktop } from "@/lib/use-media";
import { CloseIcon, DownIcon, UpIcon } from "./icons";
import { Button } from "./ui/button";
import { Input, Label, Select, Textarea } from "./ui/input";
import { Segmented } from "./ui/segmented";
import { Sheet } from "./ui/sheet";

export type RepeatChoice = "none" | "day" | "week" | "month";

export const repeatOptions: { value: RepeatChoice; label: string }[] = [
  { value: "none", label: "Does not repeat" },
  { value: "day", label: "Every day" },
  { value: "week", label: "Every week" },
  { value: "month", label: "Every month" },
];

/** The simple repeat choices until the full picker arrives (PLAN.md Phase 5). */
export function repeatFromChoice(c: RepeatChoice): Pick<Repeat, "freq" | "every" | "mode"> | null {
  return c === "none" ? null : { freq: c, every: 1, mode: "after_done" };
}

// A richer rule made elsewhere shows as its frequency; choosing again here
// replaces it with the simple form.
function choiceOf(r: Repeat | undefined): RepeatChoice {
  return !r || r.deletedAt !== null ? "none" : r.freq;
}

/** The value of the "Belongs to" select: "", "g:<id>" or "p:<id>". */
function parentValue(t: Task): string {
  if (t.projectId) return `p:${t.projectId}`;
  if (t.goalId) return `g:${t.goalId}`;
  return "";
}

export function ParentOptions() {
  const goals = useOpenGoals();
  const projects = useProjects();
  const openProjects = live(projects.values()).filter((p) => p.status === "open");
  return (
    <>
      <option value="">Nothing, standalone</option>
      {goals.length > 0 && (
        <optgroup label="Goals">
          {goals.map((g) => (
            <option key={g.id} value={`g:${g.id}`}>
              {g.title}
            </option>
          ))}
        </optgroup>
      )}
      {openProjects.length > 0 && (
        <optgroup label="Projects">
          {openProjects.map((p) => (
            <option key={p.id} value={`p:${p.id}`}>
              {p.title}
            </option>
          ))}
        </optgroup>
      )}
    </>
  );
}

export function parsedParent(v: string): { goalId: string | null; projectId: string | null } {
  if (v.startsWith("p:")) return { goalId: null, projectId: v.slice(2) };
  if (v.startsWith("g:")) return { goalId: v.slice(2), projectId: null };
  return { goalId: null, projectId: null };
}

function TaskForm({ task, onClose }: { task: Task; onClose: () => void }) {
  const actions = useActions();
  const today = useToday();
  const repeats = useRepeats();
  const goals = useGoals();
  const desktop = useIsDesktop();
  const planned = usePlanned(today);
  const [title, setTitle] = useState(task.title);
  const [notes, setNotes] = useState(task.notes);
  const [message, setMessage] = useState("");
  const id = useId();

  // Another device may change the task while it is open.
  useEffect(() => setTitle(task.title), [task.title]);
  useEffect(() => setNotes(task.notes), [task.notes]);

  async function run(fn: () => Promise<unknown>) {
    setMessage("");
    try {
      await fn();
    } catch (err) {
      if (err instanceof ActionError) setMessage(err.message);
      else throw err;
    }
  }

  const saveTitle = () => {
    const clean = title.trim();
    if (!clean) setTitle(task.title);
    else if (clean !== task.title)
      void run(() => actions.update(task.id, { title: clean.slice(0, 300) }));
  };
  const saveNotes = () => {
    if (notes !== task.notes)
      void run(() => actions.update(task.id, { notes: notes.slice(0, 20000) }));
  };

  const rank = planned.findIndex((t) => t.id === task.id);
  const goal = task.goalId ? goals.get(task.goalId) : undefined;

  return (
    <div className="flex flex-col gap-5 pb-2">
      <div>
        <Label htmlFor={`${id}-title`}>Task</Label>
        <Input
          id={`${id}-title`}
          value={title}
          maxLength={300}
          onChange={(e) => setTitle(e.target.value)}
          onBlur={saveTitle}
          onKeyDown={(e) => {
            if (e.key === "Enter") e.currentTarget.blur();
          }}
          className="text-lg font-medium"
        />
      </div>

      <div>
        <span
          className="mb-2 block text-sm font-semibold text-muted-foreground"
          id={`${id}-status`}
        >
          Status
        </span>
        <Segmented<TaskStatus>
          label="Status"
          value={task.status}
          options={[
            { value: "open", label: "Open" },
            { value: "done", label: "Done" },
            { value: "dropped", label: "Dropped" },
          ]}
          onChange={(s) =>
            void run(() =>
              actions.update(task.id, { status: s, doneAt: s === "done" ? Date.now() : null }),
            )
          }
        />
      </div>

      <div className="grid grid-cols-2 gap-3 min-[900px]:grid-cols-1">
        <div>
          <Label htmlFor={`${id}-planned`}>Planned for</Label>
          <Input
            id={`${id}-planned`}
            type="date"
            value={task.plannedOn ?? ""}
            onChange={(e) =>
              void run(() =>
                e.target.value ? actions.plan(task.id, e.target.value) : actions.unplan(task.id),
              )
            }
          />
        </div>
        <div>
          <Label htmlFor={`${id}-due`}>Due</Label>
          <Input
            id={`${id}-due`}
            type="date"
            value={task.due ?? ""}
            onChange={(e) =>
              void run(() => actions.update(task.id, { due: e.target.value || null }))
            }
          />
        </div>
      </div>
      {message && (
        <p className="-mt-2 text-sm" role="status">
          {message}
        </p>
      )}

      {!desktop && rank >= 0 && planned.length > 1 && (
        <div className="flex items-center gap-2">
          <span className="mr-auto text-sm font-semibold text-muted-foreground">
            Number {rank + 1} of today's {planned.length}
          </span>
          <Button
            variant="small"
            aria-label="Move up"
            disabled={rank === 0}
            onClick={() => void actions.move(task.id, rank - 1)}
          >
            <UpIcon size={20} />
          </Button>
          <Button
            variant="small"
            aria-label="Move down"
            disabled={rank === planned.length - 1}
            onClick={() => void actions.move(task.id, rank + 1)}
          >
            <DownIcon size={20} />
          </Button>
        </div>
      )}

      <div>
        <Label htmlFor={`${id}-parent`}>Belongs to</Label>
        <Select
          id={`${id}-parent`}
          value={parentValue(task)}
          onChange={(e) => void run(() => actions.update(task.id, parsedParent(e.target.value)))}
        >
          <ParentOptions />
          {goal && goal.status !== "open" && <option value={`g:${goal.id}`}>{goal.title}</option>}
        </Select>
      </div>

      <div>
        <Label htmlFor={`${id}-repeat`}>Repeat</Label>
        <Select
          id={`${id}-repeat`}
          value={choiceOf(task.repeatId ? repeats.get(task.repeatId) : undefined)}
          onChange={(e) =>
            void run(() =>
              actions.setRepeat(task.id, repeatFromChoice(e.target.value as RepeatChoice)),
            )
          }
        >
          {repeatOptions.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </Select>
        {task.repeatId && (
          <p className="mt-1.5 text-[13px] text-muted-foreground">
            The next one is added when you finish this one.
          </p>
        )}
      </div>

      <div>
        <Label htmlFor={`${id}-notes`}>Notes</Label>
        <Textarea
          id={`${id}-notes`}
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
          onBlur={saveNotes}
        />
      </div>

      <div className="flex justify-between">
        <Button
          variant="text"
          onClick={() =>
            void run(async () => {
              await actions.remove(task.id);
              onClose();
            })
          }
        >
          Delete task
        </Button>
        {!desktop && (
          <Button variant="text" onClick={onClose}>
            Close
          </Button>
        )}
      </div>
    </div>
  );
}

/** Phone: the open task as a bottom sheet. */
export function TaskSheet() {
  const [id, open] = useOpenTask();
  const task = useTask(id);
  const desktop = useIsDesktop();
  if (desktop) return null;
  return (
    <Sheet
      open={task !== null}
      onOpenChange={(o) => !o && open(null)}
      title={task?.title ?? "Task"}
    >
      {task && <TaskForm key={task.id} task={task} onClose={() => open(null)} />}
    </Sheet>
  );
}

/** Desktop: the open task in the right column, or null when none is open. */
export function TaskPanel() {
  const [id, open] = useOpenTask();
  const task = useTask(id);
  if (!task) return null;
  return (
    <section aria-labelledby="task-panel-title">
      <div className="mb-4 flex items-center justify-between">
        <h2 id="task-panel-title" className="text-[22px] font-bold">
          Task
        </h2>
        <Button variant="icon" aria-label="Close task" onClick={() => open(null)}>
          <CloseIcon size={20} />
        </Button>
      </div>
      <TaskForm key={task.id} task={task} onClose={() => open(null)} />
    </section>
  );
}
