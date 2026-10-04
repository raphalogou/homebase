import { NavLink } from "react-router";
import { useGoalProgress, useOpenGoals } from "@/data/hooks";
import { cn } from "@/lib/utils";

/** The goals as serif lines linking to their detail; used by Goals and beside a goal. */
export function GoalList({ compact = false }: { compact?: boolean }) {
  const goals = useOpenGoals();
  const progress = useGoalProgress();
  return (
    <ul>
      {goals.map((g) => {
        const p = progress(g.id);
        return (
          <li key={g.id} className="border-t border-line">
            <NavLink
              to={`/goals/${g.id}`}
              className={({ isActive }) =>
                cn(
                  "block py-3",
                  compact && "-mx-3 rounded-md px-3",
                  isActive && compact && "bg-soft",
                )
              }
            >
              <span
                className={cn(
                  "block font-serif",
                  compact ? "text-[17px]/[1.3]" : "text-xl/[1.25] min-[900px]:text-[21px]/[1.25]",
                )}
              >
                {g.title}
              </span>
              <span className="mt-1 block text-[13px] text-muted-foreground">
                {p.total === 0 ? "No tasks yet" : `${p.done} of ${p.total} tasks done`}
              </span>
            </NavLink>
          </li>
        );
      })}
    </ul>
  );
}
