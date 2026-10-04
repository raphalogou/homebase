import { useState } from "react";
import { Link, useNavigate } from "react-router";
import { Columns } from "@/components/app-shell";
import { CheckIcon, NoticeIcon } from "@/components/icons";
import { Empty, ScreenTitle, SectionLabel } from "@/components/section";
import { useNotify } from "@/components/toaster";
import { Button } from "@/components/ui/button";
import { Segmented } from "@/components/ui/segmented";
import { ActionError } from "@/data/hooks";
import { useCompleteReview, useReview } from "@/data/review";
import type { ReviewAction, ReviewSummary } from "@/data/types";
import { longDate } from "@/lib/dates";

function summarySentence(s: ReviewSummary): string {
  if (s.doneCount === 0) return "No tasks finished this week. A new week starts soon.";
  const total = `You finished ${s.doneCount} ${s.doneCount === 1 ? "task" : "tasks"} this week`;
  const parts = s.doneByGoal.slice(0, 3).map((g) => `${g.count} for ${g.title}`);
  if (parts.length === 0) return `${total}.`;
  const list =
    parts.length === 1
      ? parts[0]
      : `${parts.slice(0, -1).join(", ")} and ${parts[parts.length - 1]}`;
  return `${total}, ${list}.`;
}

function weeksAgo(ms: number): string {
  const days = Math.floor((Date.now() - ms) / 86_400_000);
  if (days < 14) return `${days} days ago`;
  return `${Math.floor(days / 7)} weeks ago`;
}

// DESIGN.md "Weekly review": a one-sentence summary in the serif, Gone quiet
// with Keep, Pause and Drop, Next week, and Finish review.
export default function Review() {
  const [review] = useReview();

  const main = (
    <>
      <ScreenTitle>Weekly review</ScreenTitle>
      {review.state === "loading" && (
        <p className="mt-4 text-muted-foreground">Gathering the week.</p>
      )}
      {review.state === "offline" && (
        <div className="mt-8">
          <Empty>The review needs a connection. Try again when online.</Empty>
        </div>
      )}
      {review.state === "error" && (
        <div className="mt-8">
          <Empty>{review.message}</Empty>
        </div>
      )}
      {review.state === "ready" && <ReviewBody summary={review.data} />}
    </>
  );
  return <Columns main={main} />;
}

function ReviewBody({ summary }: { summary: ReviewSummary }) {
  const complete = useCompleteReview();
  const navigate = useNavigate();
  const notify = useNotify();
  const [busy, setBusy] = useState(false);
  const [choices, setChoices] = useState<Record<string, ReviewAction>>(() =>
    Object.fromEntries(summary.quiet.map((q) => [q.id, "keep" as const])),
  );

  async function finish() {
    setBusy(true);
    try {
      await complete(summary, choices);
      notify({ title: "Review done. See you next week.", icon: CheckIcon });
      navigate("/");
    } catch (err) {
      if (err instanceof ActionError) notify({ title: err.message, icon: NoticeIcon });
      else throw err;
      setBusy(false);
    }
  }

  return (
    <>
      <p className="mt-2 text-muted-foreground">
        Week of {longDate(summary.weekStart)}
        {summary.completed ? " · already reviewed" : ""}
      </p>
      <p className="mt-6 font-serif text-[22px]/[1.35] min-[900px]:text-2xl/[1.35]">
        {summarySentence(summary)}
      </p>

      <section aria-labelledby="quiet" className="mt-10">
        <SectionLabel id="quiet">Gone quiet</SectionLabel>
        {summary.quiet.length === 0 ? (
          <Empty>Nothing has gone quiet.</Empty>
        ) : (
          <ul>
            {summary.quiet.map((q) => (
              <li key={q.id} className="border-t border-line py-3">
                <Link
                  to={q.kind === "goal" ? `/goals/${q.id}` : `/projects/${q.id}`}
                  className={
                    q.kind === "goal"
                      ? "font-serif text-xl/[1.25]"
                      : "text-[17px]/[1.3] font-medium"
                  }
                >
                  {q.title}
                </Link>
                <p className="mt-0.5 text-[13px] text-muted-foreground">
                  {q.kind === "goal" ? "Goal" : "Project"} · last active {weeksAgo(q.lastActivity)}
                </p>
                <Segmented<ReviewAction>
                  label={`What to do with ${q.title}`}
                  tone="ink"
                  className="mt-2 max-w-sm"
                  value={choices[q.id] ?? "keep"}
                  options={[
                    { value: "keep", label: "Keep" },
                    { value: "pause", label: "Pause" },
                    { value: "drop", label: "Drop" },
                  ]}
                  onChange={(a) => setChoices((c) => ({ ...c, [q.id]: a }))}
                />
              </li>
            ))}
          </ul>
        )}
      </section>

      <section aria-labelledby="next-week" className="mt-10">
        <SectionLabel id="next-week">Next week</SectionLabel>
        {summary.goalsWithNothing.length === 0 ? (
          <Empty>Every goal moved this week.</Empty>
        ) : (
          <ul>
            {summary.goalsWithNothing.map((g) => (
              <li key={g.id} className="flex items-center gap-3 border-t border-line py-3">
                <p className="min-w-0 flex-1">
                  <span className="block font-serif text-xl/[1.25]">{g.title}</span>
                  <span className="text-[13px] text-muted-foreground">Nothing done this week</span>
                </p>
                <Button variant="small" onClick={() => navigate(`/goals/${g.id}`)}>
                  Plan a task
                </Button>
              </li>
            ))}
          </ul>
        )}
      </section>

      <Button
        className="mt-10 w-full min-[900px]:w-auto"
        disabled={busy}
        onClick={() => void finish()}
      >
        Finish review
      </Button>
    </>
  );
}
