import { Columns } from "@/components/app-shell";
import { GoalLine } from "@/components/goal-line";
import { Empty, ScreenTitle } from "@/components/section";
import { useGoalProgress, useOpenGoals } from "@/data/hooks";

// The list only; goal detail and projects come in PLAN.md Phase 3.
export default function Goals() {
  const goals = useOpenGoals();
  const progress = useGoalProgress();
  return (
    <Columns
      main={
        <>
          <ScreenTitle>Goals</ScreenTitle>
          <div className="mt-8">
            {goals.length === 0 ? (
              <Empty>No goals yet.</Empty>
            ) : (
              <ul>
                {goals.map((g) => (
                  <GoalLine key={g.id} goal={g} {...progress(g.id)} />
                ))}
              </ul>
            )}
          </div>
        </>
      }
    />
  );
}
