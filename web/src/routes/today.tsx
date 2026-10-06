import { Link, useNavigate } from "react-router";
import { Columns } from "@/components/app-shell";
import { CaptureBar } from "@/components/capture-bar";
import { GoalLine, Progress } from "@/components/goal-line";
import {
  NoticeIcon,
  PlannedIcon,
  RemindersIcon,
  ReviewIcon,
  SettingsIcon,
} from "@/components/icons";
import { InstallButton } from "@/components/install-button";
import { Empty, ScreenTitle, SectionHeading, SectionLabel } from "@/components/section";
import { EmptyBlock, FirstLoad, SkeletonRows } from "@/components/states";
import { useTaskMeta } from "@/components/task-row";
import { useNotify } from "@/components/toaster";
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
  useWeekStart,
} from "@/data/hooks";
import { useReviewDue } from "@/data/review";
import { useFirstLoad } from "@/data/sync-status";
import type { Task } from "@/data/types";
import { longDate } from "@/lib/dates";
import { useOpenModal, useOpenPick } from "@/lib/modal";
import { useIsDesktop } from "@/lib/use-media";

const SLOT_TEXT = ["Choose your first", "Choose a second", "Choose a third"];

const NO_GOALS = "A goal is the reason behind your tasks. Start with one or two.";

function NoGoals() {
  return (
    <EmptyBlock
      serif
      title="No goals yet."
      action={{ label: "Add a goal", to: "/goals", text: true }}
    >
      {NO_GOALS}
    </EmptyBlock>
  );
}

// The first load after logging in: goal lines and Your three as skeletons.
function TodaySkeleton({ goals }: { goals: boolean }) {
  return (
    <>
      {goals && (
        <section className="mt-8">
          <SectionLabel>Goals</SectionLabel>
          <SkeletonRows kind="line" label="Loading your goals" />
        </section>
      )}
      <section className="mt-10">
        <SectionHeading>Your three</SectionHeading>
        <SkeletonRows label="Loading your tasks" />
      </section>
    </>
  );
}
const slotClass =
  "flex min-h-16 w-full items-center rounded-[12px] border border-dashed border-border-strong px-4 text-left font-medium text-muted-foreground";

export default function Today() {
  const today = useToday();
  const desktop = useIsDesktop();

  const main = (
    <>
      <p className="text-sm font-semibold text-muted-foreground">{longDate(today)}</p>
      <ScreenTitle className="mt-1">Today</ScreenTitle>
      <FirstLoad what="tasks" skeleton={<TodaySkeleton goals={!desktop} />}>
        <TodayBody />
      </FirstLoad>
    </>
  );

  return <Columns main={main} side={<TodaySide />} capture={!desktop} />;
}

function TodayBody() {
  const today = useToday();
  const planned = usePlanned(today);
  const goals = useOpenGoals();
  const progress = useGoalProgress();
  const actions = useActions();
  const desktop = useIsDesktop();
  const [, setPick] = useOpenPick();
  const openModal = useOpenModal();
  const room = planned.length < MAX_PER_DAY;

  return (
    <>
      {!desktop && (
        <section aria-labelledby="goals-label" className="mt-8">
          <SectionLabel id="goals-label">Goals</SectionLabel>
          {goals.length === 0 ? (
            <NoGoals />
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
        <div className="flex items-baseline justify-between gap-4">
          <SectionHeading id="three-heading">Your three</SectionHeading>
          {/* Always reachable, so a full day can still be changed. */}
          <Button variant="text" aria-label="Change your three" onClick={() => setPick(true)}>
            Change
          </Button>
        </div>
        {planned.length === 0 && (
          <p className="mb-4 border-t border-line pt-3 text-muted-foreground">
            Pick up to three things that would make today a good day.
          </p>
        )}
        <YourThree
          tasks={planned}
          grip={desktop}
          onToggle={(id) => void actions.toggleDone(id)}
          onMove={(id, to) => void actions.move(id, to)}
        />
        {room && (
          <div className={planned.length > 0 ? "border-t border-line pt-3" : ""}>
            <button type="button" className={slotClass} onClick={() => setPick(true)}>
              {SLOT_TEXT[planned.length]}
            </button>
          </div>
        )}
      </section>

      {!desktop && (
        <div className="mt-8">
          <ReviewLink />
        </div>
      )}

      {desktop ? (
        <div className="mt-10">
          <CaptureBar />
        </div>
      ) : (
        <CaptureBar fixed />
      )}
      <div className="mt-8 flex flex-wrap gap-x-5">
        {/* The phone's tab bar has four places; these are reached from here. */}
        {!desktop && (
          <>
            <button
              type="button"
              onClick={() => openModal("reminders")}
              className="inline-flex min-h-11 items-center gap-2 px-1 font-semibold underline underline-offset-[3px]"
            >
              <RemindersIcon size={20} />
              Reminders
            </button>
            <Link
              to="/settings"
              className="inline-flex min-h-11 items-center gap-2 px-1 font-semibold underline underline-offset-[3px]"
            >
              <SettingsIcon size={20} />
              Settings
            </Link>
          </>
        )}
        <InstallButton />
      </div>
    </>
  );
}

function TodaySide() {
  const today = useToday();
  const planned = usePlanned(today);
  const goals = useOpenGoals();
  const progress = useGoalProgress();
  const navigate = useNavigate();
  const { loading } = useFirstLoad();
  const room = planned.length < MAX_PER_DAY;
  if (loading)
    return (
      <section>
        <SectionLabel>Goals</SectionLabel>
        <SkeletonRows kind="line" count={3} label="Loading your goals" />
      </section>
    );

  return (
    <>
      <section aria-labelledby="side-goals">
        <SectionLabel id="side-goals">Goals</SectionLabel>
        {goals.length === 0 ? (
          <NoGoals />
        ) : (
          <ul>
            {goals.map((g) => {
              const p = progress(g.id);
              return (
                <li key={g.id} className="border-t border-line">
                  <Link to={`/goals/${g.id}`} className="block py-3">
                    <span className="block font-serif text-[19px]/[1.25]">{g.title}</span>
                    <Progress
                      done={p.done}
                      total={p.total}
                      label={`${g.title}: ${p.done} of ${p.total} done`}
                    />
                  </Link>
                </li>
              );
            })}
          </ul>
        )}
      </section>
      <section aria-labelledby="side-week" className="mt-10">
        <SectionLabel id="side-week">This week</SectionLabel>
        <WeekStrip selected={null} onSelect={(d) => d && navigate(`/plan?day=${d}`)} />
        <div className="mt-4">
          <ReviewLink />
        </div>
      </section>
      <Suggestions room={room} />
    </>
  );
}

function Suggestions({ room }: { room: boolean }) {
  const s = useSuggestions();
  const groups: [string, Task[]][] = [
    ["Due soon", s.dueSoon],
    ["In progress", s.inProgress],
    ["Moved a few times", s.moved],
  ];
  const any = groups.some(([, list]) => list.length > 0);
  return (
    <section aria-labelledby="suggest-heading" className="mt-10">
      <h2 id="suggest-heading" className="mb-1 text-[22px] font-bold">
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
  const notify = useNotify();
  return (
    <li className="flex min-h-16 items-center gap-3 border-t border-line py-2.5">
      <div className="min-w-0 flex-1">
        <p className="text-[17px]/[1.3] font-medium">{task.title}</p>
        {meta && <p className="mt-0.5 text-[13px] text-muted-foreground">{meta}</p>}
      </div>
      <Button
        variant="small"
        disabled={!room}
        aria-label={`Plan for today: ${task.title}`}
        onClick={async () => {
          try {
            await actions.plan(task.id, today);
            notify({ title: `Planned for today: ${task.title}`, icon: PlannedIcon });
          } catch (err) {
            if (err instanceof ActionError) notify({ title: err.message, icon: NoticeIcon });
            else throw err;
          }
        }}
      >
        Today
      </Button>
    </li>
  );
}

/** From Friday until this week's review is done (docs/SPEC.md section 5). */
function ReviewLink() {
  const today = useToday();
  const weekStart = useWeekStart();
  const due = useReviewDue(today, weekStart);
  if (!due) return null;
  return (
    <Link
      to="/review"
      className="flex min-h-16 items-center gap-3 rounded-[12px] border border-border-strong bg-field px-4"
    >
      <ReviewIcon size={20} className="shrink-0" />
      <span className="min-w-0 flex-1">
        <span className="block font-semibold">Weekly review</span>
        <span className="block text-[13px] text-muted-foreground">
          About a minute: keep, pause or drop what went quiet.
        </span>
      </span>
    </Link>
  );
}
