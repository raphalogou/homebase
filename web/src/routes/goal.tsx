import { useParams } from "react-router";
import { AddField } from "@/components/add-field";
import { Columns } from "@/components/app-shell";
import { Attachments } from "@/components/attachments";
import { BackLink } from "@/components/back-link";
import { Progress } from "@/components/goal-line";
import { GoalList } from "@/components/goal-list";
import { NotesField } from "@/components/notes-field";
import { ProjectRow } from "@/components/project-row";
import { Empty, SectionLabel } from "@/components/section";
import { StatusAndDelete } from "@/components/status-and-delete";
import { TaskRow } from "@/components/task-row";
import { TitleField } from "@/components/title-field";
import { useActions, useGoalProgress, useTasks } from "@/data/hooks";
import { useGoal, useGoalTasks, useProjectList, useStructureActions } from "@/data/structure";
import { useMedia } from "@/lib/use-media";

export default function GoalScreen() {
  const { id } = useParams();
  const goal = useGoal(id);
  const progress = useGoalProgress();
  const tasks = useTasks();
  const projects = useProjectList();
  const own = useGoalTasks(id ?? "");
  const actions = useActions();
  const structure = useStructureActions();
  const wide = useMedia("(min-width: 1100px)");

  if (!goal) {
    return (
      <Columns
        main={
          <>
            <BackLink to="/goals">Goals</BackLink>
            <div className="mt-6">
              <Empty>This goal is not here. It may have been deleted.</Empty>
            </div>
          </>
        }
      />
    );
  }

  const p = progress(goal.id);
  const goalProjects = projects.filter((x) => x.goalId === goal.id && x.status !== "dropped");

  const main = (
    <>
      <BackLink to="/goals">Goals</BackLink>
      <h1 className="mt-3">
        <TitleField
          label="Goal"
          value={goal.title}
          onSave={(title) => void structure.updateGoal(goal.id, { title })}
          className="font-serif text-[32px]/[1.15] font-medium min-[900px]:text-[44px]/[1.1]"
        />
      </h1>
      <Progress done={p.done} total={p.total} label={`${p.done} of ${p.total} tasks done`} />
      <p className="mt-1.5 text-[13px] text-muted-foreground">
        {p.total ? `${p.done} of ${p.total} tasks done` : "No tasks yet"}
      </p>

      <section aria-labelledby="goal-projects" className="mt-10">
        <SectionLabel id="goal-projects">Projects</SectionLabel>
        {goalProjects.length > 0 && (
          <ul className="mb-3">
            {goalProjects.map((x) => (
              <ProjectRow key={x.id} project={x} tasks={tasks} />
            ))}
          </ul>
        )}
        <AddField
          label="New project for this goal"
          placeholder="Add a project"
          button="Add project"
          onAdd={(title) => structure.addProject(title, goal.id)}
        />
      </section>

      <section aria-labelledby="goal-tasks" className="mt-10">
        <SectionLabel id="goal-tasks">Tasks on their own</SectionLabel>
        <AddField
          label="New task for this goal"
          placeholder="Add a task"
          button="Add task"
          onAdd={(title) => actions.addTask({ title, goalId: goal.id })}
        />
        <ul className="mt-3">
          {own.map((t) => (
            <li key={t.id}>
              <TaskRow task={t} hideContext onToggle={() => void actions.toggleDone(t.id)} />
            </li>
          ))}
        </ul>
      </section>

      <div className="mt-10">
        <NotesField
          value={goal.notes}
          onSave={(notes) => structure.updateGoal(goal.id, { notes })}
        />
      </div>

      <div className="mt-10">
        <Attachments owner={{ kind: "goal", id: goal.id }} heading="Links" />
      </div>

      <StatusAndDelete
        kind="goal"
        id={goal.id}
        title={goal.title}
        status={goal.status}
        after="/goals"
      />
    </>
  );

  // Desktop shows the goal list on the left (DESIGN.md "Goal").
  if (!wide) return <Columns main={main} />;
  return (
    <div className="flex">
      <nav
        aria-label="Goals"
        className="sticky top-0 h-dvh w-[260px] shrink-0 overflow-y-auto border-r border-line px-6 pt-12"
      >
        <SectionLabel>Goals</SectionLabel>
        <GoalList compact />
      </nav>
      <div className="min-w-0 flex-1">
        <Columns main={main} />
      </div>
    </div>
  );
}
