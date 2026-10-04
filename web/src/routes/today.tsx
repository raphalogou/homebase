import { type RefObject, useRef, useState } from "react";
import { Link } from "react-router";
import { Columns } from "@/components/app-shell";
import { CaptureBar } from "@/components/capture-bar";
import { GoalLine, Progress } from "@/components/goal-line";
import { InstallButton } from "@/components/install-button";
import { Empty, ScreenTitle, SectionHeading, SectionLabel } from "@/components/section";
import { useTaskMeta } from "@/components/task-row";
import { Button } from "@/components/ui/button";
import { WeekStrip } from "@/components/week-strip";
import { YourThree } from "@/components/your-three";
import {
  ActionError,
  MAX_PER_DAY,
  useActions,
  useGoalProgress,
  useOpenGoals,
  usePlanned,
  useSuggestions,
  useToday,
} from "@/data/hooks";
import type { Task } from "@/data/types";
import { longDate } from "@/lib/dates";
import { useIsDesktop } from "@/lib/use-media";

const SLOT_TEXT = ["Choose your three", "Choose a second", "Choose a third"];
const slotClass =
  "flex min-h-16 w-full items-center rounded-[12px] border border-dashed [border-color:var(--dashed)] px-4 text-left font-medium text-muted-foreground";

export default function Today() {
  const today = useToday();
  const planned = usePlanned(today);
  const goals = useOpenGoals();
  const progress = useGoalProgress();
  const actions = useActions();
  const desktop = useIsDesktop();
  const suggestionsRef = useRef<HTMLHeadingElement>(null);
  const room = planned.length < MAX_PER_DAY;

  const main = (
    <>
      <p className="text-sm font-semibold text-muted-foreground">{longDate(today)}</p>
      <ScreenTitle className="mt-1">Today</ScreenTitle>

      {!desktop && (
        <section aria-labelledby="goals-label" className="mt-8">
          <SectionLabel id="goals-label">Goals</SectionLabel>
          {goals.length === 0 ? (
            <Empty>No goals yet.</Empty>
          ) : (
            <ul>
              {goals.map((g) => (
                <GoalLine key={g.id} goal={g} {...progress(g.id)} />
              ))}
            </ul>
          )}
        </section>
      )}

      <section aria-labelledby="three-heading" className="mt-10">
        <SectionHeading id="three-heading">Your three</SectionHeading>
        <YourThree
          tasks={planned}
          grip={desktop}
          onToggle={(id) => void actions.toggleDone(id)}
          onMove={(id, to) => void actions.move(id, to)}
        />
        {room && (
          <div className={planned.length > 0 ? "border-t border-line pt-3" : ""}>
            {desktop ? (
              <button
                type="button"
                className={slotClass}
                onClick={() => {
                  suggestionsRef.current?.scrollIntoView({ block: "start" });
                  suggestionsRef.current?.focus();
                }}
              >
                {SLOT_TEXT[planned.length]}
              </button>
            ) : (
              <Link to="/pick" className={slotClass}>
                {SLOT_TEXT[planned.length]}
              </Link>
            )}
          </div>
        )}
      </section>

      {desktop ? (
        <div className="mt-10">
          <CaptureBar />
        </div>
      ) : (
        <CaptureBar fixed />
      )}
      <InstallButton className="mt-8" />
    </>
  );

  const side = (
    <>
      <section aria-labelledby="side-goals">
        <SectionLabel id="side-goals">Goals</SectionLabel>
        {goals.length === 0 ? (
          <Empty>No goals yet.</Empty>
        ) : (
          <ul>
            {goals.map((g) => {
              const p = progress(g.id);
              return (
                <li key={g.id} className="border-t border-line py-3">
                  <p className="font-serif text-[19px]/[1.25]">{g.title}</p>
                  <Progress
                    done={p.done}
                    total={p.total}
                    label={`${g.title}: ${p.done} of ${p.total} done`}
                  />
                </li>
              );
            })}
          </ul>
        )}
      </section>
      <section aria-labelledby="side-week" className="mt-10">
        <SectionLabel id="side-week">This week</SectionLabel>
        <WeekStrip selected={today} />
      </section>
      <Suggestions headingRef={suggestionsRef} room={room} />
    </>
  );

  return <Columns main={main} side={side} capture={!desktop} />;
}

function Suggestions({
  headingRef,
  room,
}: {
  headingRef: RefObject<HTMLHeadingElement | null>;
  room: boolean;
}) {
  const s = useSuggestions();
  const groups: [string, Task[]][] = [
    ["Due soon", s.dueSoon],
    ["In progress", s.inProgress],
    ["Moved a few times", s.moved],
  ];
  const any = groups.some(([, list]) => list.length > 0);
  return (
    <section aria-labelledby="suggest-heading" className="mt-10">
      <h2
        id="suggest-heading"
        ref={headingRef}
        tabIndex={-1}
        className="mb-1 text-[22px] font-bold"
      >
        Choose up to three
      </h2>
      {!room && (
        <p className="mb-2 text-sm text-muted-foreground">
          Today is full. Finish or move one to add another.
        </p>
      )}
      {!any && <Empty>Nothing to suggest. Plan a task from Plan or the Inbox.</Empty>}
      {groups.map(([label, list]) =>
        list.length === 0 ? null : (
          <div key={label} className="mt-5">
            <SectionLabel>{label}</SectionLabel>
            <ul>
              {list.map((t) => (
                <SuggestionRow key={t.id} task={t} room={room} />
              ))}
            </ul>
          </div>
        ),
      )}
    </section>
  );
}

function SuggestionRow({ task, room }: { task: Task; room: boolean }) {
  const actions = useActions();
  const today = useToday();
  const meta = useTaskMeta()(task);
  const [message, setMessage] = useState("");
  return (
    <li className="flex min-h-16 items-center gap-3 border-t border-line py-2.5">
      <div className="min-w-0 flex-1">
        <p className="text-[17px]/[1.3] font-medium">{task.title}</p>
        {meta && <p className="mt-0.5 text-[13px] text-muted-foreground">{meta}</p>}
        {message && <p className="mt-0.5 text-[13px]">{message}</p>}
      </div>
      <Button
        variant="small"
        disabled={!room}
        aria-label={`Plan for today: ${task.title}`}
        onClick={async () => {
          try {
            await actions.plan(task.id, today);
          } catch (err) {
            if (err instanceof ActionError) setMessage(err.message);
            else throw err;
          }
        }}
      >
        Today
      </Button>
    </li>
  );
}
