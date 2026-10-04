import type { Goal } from "@/data/types";

// Goals read as statements of intent: serif, with one meta line.
export function GoalLine({ goal, done, total }: { goal: Goal; done: number; total: number }) {
  return (
    <li className="border-t border-line py-3">
      <p className="font-serif text-xl/[1.25] min-[900px]:text-[21px]/[1.25]">{goal.title}</p>
      <p className="mt-1 text-[13px] text-muted-foreground">
        {total === 0 ? "No tasks yet" : `${done} of ${total} tasks done`}
      </p>
    </li>
  );
}

/** The 4 px progress line of a goal: --line track, --ink fill. */
export function Progress({ done, total, label }: { done: number; total: number; label: string }) {
  const pct = total === 0 ? 0 : Math.round((done / total) * 100);
  return (
    <div
      role="progressbar"
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={pct}
      className="mt-2 h-1 w-full rounded-full bg-line"
    >
      <div className="h-1 rounded-full bg-ink" style={{ width: `${pct}%` }} />
    </div>
  );
}
