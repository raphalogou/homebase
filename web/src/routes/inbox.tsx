import { useId, useState } from "react";
import { useNavigate } from "react-router";
import { Columns } from "@/components/app-shell";
import { CaptureBar } from "@/components/capture-bar";
import { DoneBox } from "@/components/checkbox";
import {
  CheckIcon,
  GoalsIcon,
  MakeProjectIcon,
  NoticeIcon,
  PlanIcon,
  PlannedIcon,
  TodayIcon,
} from "@/components/icons";
import { Empty, ScreenTitle, SectionHeading } from "@/components/section";
import { TaskRow } from "@/components/task-row";
import { useNotify } from "@/components/toaster";
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
import { dayHeading } from "@/lib/dates";
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
  const navigate = useNavigate();
  const notify = useNotify();
  const [, open] = useOpenTask();
  const [panel, setPanel] = useState<Panel>(null);
  const id = useId();

  async function run(fn: () => Promise<unknown>) {
    try {
      await fn();
    } catch (err) {
      if (err instanceof ActionError) notify({ title: err.message, icon: NoticeIcon });
      else throw err;
    }
  }

  // Approved: verbs that say what happens, each with its icon.
  return (
    <li className="border-t border-line py-3">
      <div className="flex items-center gap-3">
        <DoneBox
          done={task.status === "done"}
          title={task.title}
          onToggle={() =>
            void run(async () => {
              // A done task leaves the Inbox, so say where it went.
              await actions.toggleDone(task.id);
              notify({
                title: `Done: ${task.title}`,
                icon: CheckIcon,
                action: { label: "Undo", run: () => void actions.toggleDone(task.id) },
              });
            })
          }
        />
        <button
          type="button"
          onClick={() => open(task.id)}
          className="min-h-11 min-w-0 flex-1 text-left text-[17px]/[1.3] font-medium min-[900px]:text-lg/[1.3]"
        >
          {task.title}
        </button>
      </div>
      {/* On wider screens, indented to line up with the title; the phone needs the width. */}
      <fieldset className="mt-2 flex min-w-0 flex-wrap gap-2 min-[900px]:pl-[47px]">
        <legend className="sr-only">Place {task.title}</legend>
        <Button
          variant="chip"
          onClick={() =>
            void run(async () => {
              await actions.plan(task.id, today);
              notify({ title: `Planned for today: ${task.title}`, icon: PlannedIcon });
            })
          }
        >
          <TodayIcon size={18} />
          Do today
        </Button>
        <Button
          variant="chip"
          aria-expanded={panel === "date"}
          onClick={() => setPanel(panel === "date" ? null : "date")}
        >
          <PlanIcon size={18} />
          Pick a day
        </Button>
        <Button
          variant="chip"
          aria-expanded={panel === "goal"}
          onClick={() => setPanel(panel === "goal" ? null : "goal")}
        >
          <GoalsIcon size={18} />
          Add to a goal
        </Button>
        <Button
          variant="chip"
          onClick={() =>
            void run(async () => {
              const projectId = await actions.promote(task.id, null);
              notify({
                title: `Made a project: ${task.title}`,
                icon: MakeProjectIcon,
                action: { label: "Open", run: () => navigate(`/projects/${projectId}`) },
              });
            })
          }
        >
          <MakeProjectIcon size={18} />
          Make it a project
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
              if (day)
                void run(async () => {
                  await actions.plan(task.id, day);
                  notify({
                    title: `Planned for ${dayHeading(day, today)}: ${task.title}`,
                    icon: PlannedIcon,
                  });
                });
            }}
          />
        </div>
      )}
      {panel === "goal" && (
        <div className="mt-3 max-w-80">
          <Label htmlFor={`${id}-goal`}>Goal</Label>
          {goals.length === 0 ? (
            <p className="text-muted-foreground">No goals yet. Add one on the Goals screen.</p>
          ) : (
            <Select
              id={`${id}-goal`}
              defaultValue=""
              onChange={(e) => {
                const goalId = e.target.value;
                const goal = goals.find((g) => g.id === goalId);
                if (goalId)
                  void run(async () => {
                    await actions.update(task.id, { goalId, projectId: null });
                    notify({ title: `Added to ${goal?.title ?? "the goal"}`, icon: GoalsIcon });
                  });
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
    </li>
  );
}
