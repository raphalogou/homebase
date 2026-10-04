import { AddField } from "@/components/add-field";
import { Columns } from "@/components/app-shell";
import { GoalList } from "@/components/goal-list";
import { Empty, ScreenTitle } from "@/components/section";
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
            {goals.length === 0 ? <Empty>No goals yet. Write one below.</Empty> : <GoalList />}
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
