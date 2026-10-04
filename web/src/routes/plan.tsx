import { type FormEvent, useId, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { Columns } from "@/components/app-shell";
import { AddIcon, CheckIcon, NoticeIcon } from "@/components/icons";
import { ScreenTitle, SectionHeading, SectionLabel } from "@/components/section";
import { EmptyBlock, FirstLoad, SkeletonRows } from "@/components/states";
import {
  ParentOptions,
  parsedParent,
  type RepeatChoice,
  repeatFromChoice,
  repeatOptions,
} from "@/components/task-editor";
import { TaskRow } from "@/components/task-row";
import { useNotify } from "@/components/toaster";
import { Button } from "@/components/ui/button";
import { Input, Label, Select } from "@/components/ui/input";
import { Segmented } from "@/components/ui/segmented";
import { Sheet } from "@/components/ui/sheet";
import { WeekStrip } from "@/components/week-strip";
import { ActionError, useActions, useTasks, useToday, useWeekStart } from "@/data/hooks";
import { dayOf } from "@/data/rules";
import type { LocalDate, Task } from "@/data/types";
import { addDays, dayHeading, isDate, longDate, startOfWeek } from "@/lib/dates";
import { useIsDesktop } from "@/lib/use-media";

type Filter = "all" | "week" | "standalone" | "repeating";

const FILTERS: { value: Filter; label: string }[] = [
  { value: "all", label: "Everything" },
  { value: "week", label: "Due this week" },
  { value: "standalone", label: "Standalone" },
  { value: "repeating", label: "Repeating" },
];

interface Group {
  key: string;
  heading: string;
  tasks: Task[];
}

export default function Plan() {
  const tasks = useTasks();
  const today = useToday();
  const weekStart = useWeekStart();
  const actions = useActions();
  const navigate = useNavigate();
  const [filter, setFilter] = useState<Filter>("all");
  const [adding, setAdding] = useState(false);
  const desktop = useIsDesktop();
  const [params, setParams] = useSearchParams();
  const dayParam = params.get("day");
  const day: LocalDate | null = dayParam && isDate(dayParam) ? dayParam : null;
  const setDay = (d: LocalDate | null) =>
    setParams(
      (p) => {
        const copy = new URLSearchParams(p);
        if (d) copy.set("day", d);
        else copy.delete("day");
        return copy;
      },
      { replace: true },
    );

  const groups = useMemo(() => {
    const weekEnd = addDays(startOfWeek(today, weekStart), 6);
    const visible = tasks.filter((t) => {
      // Open tasks, plus today's finished ones so the day reads complete.
      if (t.status === "dropped") return false;
      if (t.status === "done" && t.plannedOn !== today) return false;
      if (day && t.plannedOn !== day && t.due !== day) return false;
      switch (filter) {
        case "week":
          return t.due !== null && t.due <= weekEnd;
        case "standalone":
          return t.projectId === null && t.goalId === null;
        case "repeating":
          return t.repeatId !== null;
        default:
          return true;
      }
    });

    const byDay = new Map<string, Task[]>();
    const repeating: Task[] = [];
    const undated: Task[] = [];
    for (const t of visible) {
      const d = dayOf(t);
      if (t.repeatId && filter !== "repeating") repeating.push(t);
      else if (d === null) undated.push(t);
      else byDay.set(d, [...(byDay.get(d) ?? []), t]);
    }
    const out: Group[] = [...byDay.entries()]
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([d, list]) => ({
        key: d,
        heading: d < today ? `${dayHeading(d, today)}, overdue` : dayHeading(d, today),
        tasks: list.sort(
          (a, b) => (a.planRank ?? 99) - (b.planRank ?? 99) || a.createdAt - b.createdAt,
        ),
      }));
    if (undated.length) out.push({ key: "none", heading: "No date", tasks: undated });
    if (repeating.length) out.push({ key: "repeat", heading: "Repeating", tasks: repeating });
    return out;
  }, [tasks, today, weekStart, filter, day]);

  const empty = day ? (
    <EmptyBlock title="Nothing on this day." />
  ) : tasks.length > 0 ? (
    <EmptyBlock title="No tasks match this filter." />
  ) : desktop ? (
    <EmptyBlock title="No tasks here.">
      Type one in the New task form, or capture it from Today.
    </EmptyBlock>
  ) : (
    <EmptyBlock title="Nothing planned.">Use the plus to add a task.</EmptyBlock>
  );

  const main = (
    <>
      <div className="flex items-start justify-between gap-4">
        <ScreenTitle>Plan</ScreenTitle>
        {/* The phone has no right column, so the New task form opens in a sheet. */}
        {!desktop && (
          <Button variant="icon" aria-label="Add task" onClick={() => setAdding(true)}>
            <AddIcon />
          </Button>
        )}
      </div>
      <Segmented
        label="Show"
        className="mt-6 max-w-80"
        value="tasks"
        options={[
          { value: "tasks", label: "Tasks" },
          { value: "projects", label: "Projects" },
        ]}
        onChange={(v) => v === "projects" && navigate("/plan/projects")}
      />
      <div className="mt-6">
        <WeekStrip selected={day} onSelect={setDay} shortDays />
      </div>
      <fieldset className="mt-4 -mx-6 flex min-w-0 gap-2 overflow-x-auto px-6 pb-1 [scrollbar-width:none]">
        <legend className="sr-only">Show</legend>
        {FILTERS.map((f) => (
          <Button
            key={f.value}
            variant="chip"
            aria-pressed={filter === f.value}
            onClick={() => setFilter(f.value)}
          >
            {f.label}
          </Button>
        ))}
      </fieldset>

      {day && (
        <div className="mt-6 flex items-baseline justify-between gap-4">
          <h2 className="text-[22px] font-bold">{longDate(day)}</h2>
          <Button variant="text" onClick={() => setDay(null)}>
            Show all
          </Button>
        </div>
      )}
      <div className="mt-6">
        <FirstLoad what="tasks" skeleton={<SkeletonRows count={3} label="Loading your tasks" />}>
          {groups.length === 0 && empty}
          {groups.map((g) => (
            <section key={g.key} aria-label={g.heading} className="mb-8">
              <SectionLabel>{g.heading}</SectionLabel>
              <ul>
                {g.tasks.map((t) => (
                  <li key={t.id}>
                    <TaskRow task={t} onToggle={() => void actions.toggleDone(t.id)} />
                  </li>
                ))}
              </ul>
            </section>
          ))}
        </FirstLoad>
      </div>
      {!desktop && (
        <Sheet open={adding} onOpenChange={setAdding} title="New task">
          <NewTaskForm onAdded={() => setAdding(false)} />
        </Sheet>
      )}
    </>
  );

  return <Columns main={main} side={<NewTaskForm />} />;
}

function NewTaskForm({ onAdded }: { onAdded?: () => void }) {
  const actions = useActions();
  const id = useId();
  const [title, setTitle] = useState("");
  const [parent, setParent] = useState("");
  const [day, setDay] = useState("");
  const [repeat, setRepeat] = useState<RepeatChoice>("none");
  const notify = useNotify();

  async function submit(e: FormEvent) {
    e.preventDefault();
    try {
      await actions.addTask({
        title,
        ...parsedParent(parent),
        plannedOn: day || null,
        repeat: repeatFromChoice(repeat),
      });
      setTitle("");
      setDay("");
      setRepeat("none");
      notify({ title: `Added “${title.trim()}”`, icon: CheckIcon });
      onAdded?.();
    } catch (err) {
      if (err instanceof ActionError) notify({ title: err.message, icon: NoticeIcon });
      else throw err;
    }
  }

  return (
    <section aria-labelledby={`${id}-h`}>
      <SectionHeading id={`${id}-h`}>New task</SectionHeading>
      <form onSubmit={submit} className="flex flex-col gap-4">
        <div>
          <Label htmlFor={`${id}-title`}>Task</Label>
          <Input
            id={`${id}-title`}
            data-capture
            value={title}
            maxLength={300}
            onChange={(e) => setTitle(e.target.value)}
            required
          />
        </div>
        <div>
          <Label htmlFor={`${id}-parent`}>Goal or project</Label>
          <Select id={`${id}-parent`} value={parent} onChange={(e) => setParent(e.target.value)}>
            <ParentOptions />
          </Select>
        </div>
        <div>
          <Label htmlFor={`${id}-day`}>Plan for</Label>
          <Input
            id={`${id}-day`}
            type="date"
            value={day}
            onChange={(e) => setDay(e.target.value)}
          />
        </div>
        <div>
          <Label htmlFor={`${id}-repeat`}>Repeat</Label>
          <Select
            id={`${id}-repeat`}
            value={repeat}
            onChange={(e) => setRepeat(e.target.value as RepeatChoice)}
          >
            {repeatOptions.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </Select>
        </div>
        <Button type="submit">Add task</Button>
      </form>
    </section>
  );
}
