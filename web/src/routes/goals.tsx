import { AddField } from "@/components/add-field";
import { Columns } from "@/components/app-shell";
import { GoalList } from "@/components/goal-list";
import { ScreenTitle } from "@/components/section";
import { EmptyBlock, FirstLoad, SkeletonRows } from "@/components/states";
import { useOpenGoals } from "@/data/hooks";
import { useStructureActions } from "@/data/structure";

export default function Goals() {
  const goals = useOpenGoals();
  const actions = useStructureActions();
  return (
    <Columns
      main={
        <>
          <ScreenTitle>Goals</ScreenTitle>
          <div className="mt-8">
            <FirstLoad
              what="goals"
              skeleton={<SkeletonRows kind="line" count={3} label="Loading your goals" />}
            >
              {goals.length === 0 ? (
                <EmptyBlock serif title="No goals yet.">
                  A goal is the reason behind your tasks. Start with one or two.
                </EmptyBlock>
              ) : (
                <GoalList />
              )}
            </FirstLoad>
          </div>
          {/* Approved design: the new goal field sits under the list, in the serif. */}
          <div className="mt-10">
            <AddField
              label="New goal"
              showLabel
              serif
              placeholder="A statement of intent"
              button="Add"
              onAdd={(title) => actions.addGoal(title)}
            />
          </div>
        </>
      }
    />
  );
}
