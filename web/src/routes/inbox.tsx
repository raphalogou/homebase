import { useId, useState } from "react";
import { Columns } from "@/components/app-shell";
import { CaptureBar } from "@/components/capture-bar";
import { Empty, ScreenTitle, SectionHeading } from "@/components/section";
import { TaskRow } from "@/components/task-row";
import { Button } from "@/components/ui/button";
import { Input, Label, Select } from "@/components/ui/input";
import {
  ActionError,
  useActions,
  useInbox,
  useOpenGoals,
  usePlanned,
  useToday,
} from "@/data/hooks";
import type { Task } from "@/data/types";
import { useOpenTask } from "@/lib/open-task";
import { useIsDesktop } from "@/lib/use-media";

export default function Inbox() {
  const items = useInbox();
  const desktop = useIsDesktop();
  const today = useToday();
  const planned = usePlanned(today);
  const actions = useActions();

  const main = (
    <>
      <ScreenTitle>Inbox</ScreenTitle>
      <p className="mt-2 text-muted-foreground">Captured, not yet placed</p>
      {desktop && (
        <div className="mt-8">
          <CaptureBar />
        </div>
      )}
      <ul className="mt-6">
        {items.map((t) => (
          <InboxItem key={t.id} task={t} />
        ))}
      </ul>
      {items.length === 0 && <Empty>Nothing waiting. New tasks you capture land here.</Empty>}
      {!desktop && <CaptureBar fixed />}
    </>
  );

  const side = (
    <section aria-labelledby="inbox-three">
      <SectionHeading id="inbox-three">Your three</SectionHeading>
      {planned.length === 0 ? (
        <Empty>Nothing planned for today yet.</Empty>
      ) : (
        <ol>
          {planned.map((t) => (
            <li key={t.id}>
              <TaskRow task={t} onToggle={() => void actions.toggleDone(t.id)} />
            </li>
          ))}
        </ol>
      )}
    </section>
  );

  return <Columns main={main} side={side} capture={!desktop} />;
}

type Panel = "date" | "goal" | null;

function InboxItem({ task }: { task: Task }) {
  const actions = useActions();
  const today = useToday();
  const goals = useOpenGoals();
  const [, open] = useOpenTask();
  const [panel, setPanel] = useState<Panel>(null);
  const [message, setMessage] = useState("");
  const id = useId();

  async function run(fn: () => Promise<unknown>) {
    setMessage("");
    try {
      await fn();
    } catch (err) {
      if (err instanceof ActionError) setMessage(err.message);
      else throw err;
    }
  }

  return (
    <li className="border-t border-line py-3">
      <button
        type="button"
        onClick={() => open(task.id)}
        className="min-h-11 w-full text-left text-[17px]/[1.3] font-medium min-[900px]:text-lg/[1.3]"
      >
        {task.title}
      </button>
      <fieldset className="mt-2 flex min-w-0 flex-wrap gap-2">
        <legend className="sr-only">Place {task.title}</legend>
        <Button variant="chip" onClick={() => void run(() => actions.plan(task.id, today))}>
          Today
        </Button>
        <Button
          variant="chip"
          aria-pressed={panel === "date"}
          aria-expanded={panel === "date"}
          onClick={() => setPanel(panel === "date" ? null : "date")}
        >
          Date
        </Button>
        <Button
          variant="chip"
          aria-pressed={panel === "goal"}
          aria-expanded={panel === "goal"}
          onClick={() => setPanel(panel === "goal" ? null : "goal")}
        >
          Goal
        </Button>
        <Button variant="chip" onClick={() => void run(() => actions.promote(task.id, null))}>
          Project
        </Button>
      </fieldset>

      {panel === "date" && (
        <div className="mt-3 max-w-60">
          <Label htmlFor={`${id}-date`}>Plan for</Label>
          <Input
            id={`${id}-date`}
            type="date"
            min={today}
            onChange={(e) => {
              const day = e.target.value;
              if (day) void run(() => actions.plan(task.id, day));
            }}
          />
        </div>
      )}
      {panel === "goal" && (
        <div className="mt-3 max-w-80">
          <Label htmlFor={`${id}-goal`}>Goal</Label>
          {goals.length === 0 ? (
            <p className="text-muted-foreground">No goals yet.</p>
          ) : (
            <Select
              id={`${id}-goal`}
              defaultValue=""
              onChange={(e) => {
                const goalId = e.target.value;
                if (goalId) void run(() => actions.update(task.id, { goalId, projectId: null }));
              }}
            >
              <option value="" disabled>
                Choose a goal
              </option>
              {goals.map((g) => (
                <option key={g.id} value={g.id}>
                  {g.title}
                </option>
              ))}
            </Select>
          )}
        </div>
      )}
      {message && (
        <p className="mt-2 text-sm" role="status">
          {message}
        </p>
      )}
    </li>
  );
}
