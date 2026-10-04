import { useId, useState } from "react";
import { useNavigate } from "react-router";
import { AddField } from "@/components/add-field";
import { Columns } from "@/components/app-shell";
import { GoalsIcon } from "@/components/icons";
import { ProjectRow } from "@/components/project-row";
import { Empty, ScreenTitle, SectionHeading } from "@/components/section";
import { Label, Select } from "@/components/ui/input";
import { Segmented } from "@/components/ui/segmented";
import { useOpenGoals, useTasks } from "@/data/hooks";
import { useProjectList, useStructureActions } from "@/data/structure";
import type { Project } from "@/data/types";
import { useIsDesktop } from "@/lib/use-media";

// Projects grouped under their goal, each with progress and its next task.
// The phone version was not drawn; it is the desktop list in one column.
export default function PlanProjects() {
  const goals = useOpenGoals();
  const projects = useProjectList().filter((p) => p.status !== "dropped");
  const tasks = useTasks();
  const desktop = useIsDesktop();
  const navigate = useNavigate();

  const order = (a: Project, b: Project) =>
    Number(a.status === "done") - Number(b.status === "done") || a.createdAt - b.createdAt;
  const groups = goals
    .map((g) => ({
      key: g.id,
      title: g.title,
      list: projects.filter((p) => p.goalId === g.id).sort(order),
    }))
    .filter((g) => g.list.length > 0);
  const openGoalIds = new Set(goals.map((g) => g.id));
  const loose = projects.filter((p) => p.goalId === null || !openGoalIds.has(p.goalId)).sort(order);
  if (loose.length) groups.push({ key: "none", title: "", list: loose });

  const main = (
    <>
      <ScreenTitle>Plan</ScreenTitle>
      <Segmented
        label="Show"
        className="mt-6 max-w-80"
        value="projects"
        options={[
          { value: "tasks", label: "Tasks" },
          { value: "projects", label: "Projects" },
        ]}
        onChange={(v) => v === "tasks" && navigate("/plan")}
      />
      <div className="mt-8">
        {groups.length === 0 && (
          <Empty>No projects yet. Add one to group the tasks of a bigger piece of work.</Empty>
        )}
        {groups.map((g) => (
          <section key={g.key} aria-label={g.title || "Projects without a goal"} className="mb-10">
            <h2
              className={
                g.title
                  ? "mb-2 flex items-center gap-2.5 font-serif text-xl/[1.25]"
                  : "mb-2 text-sm font-semibold text-muted-foreground"
              }
            >
              {g.title && <GoalsIcon size={20} className="shrink-0" />}
              {g.title || "Without a goal"}
            </h2>
            <ul>
              {g.list.map((p) => (
                <ProjectRow key={p.id} project={p} tasks={tasks} />
              ))}
            </ul>
          </section>
        ))}
      </div>
      {!desktop && <AddProjectForm />}
    </>
  );
  return <Columns main={main} side={<AddProjectForm />} />;
}

function AddProjectForm() {
  const goals = useOpenGoals();
  const actions = useStructureActions();
  const [goalId, setGoalId] = useState("");
  const id = useId();
  return (
    <section aria-labelledby={`${id}-h`} className="flex flex-col gap-4">
      <SectionHeading id={`${id}-h`} className="mb-0">
        Add a project
      </SectionHeading>
      <div>
        <Label htmlFor={`${id}-goal`}>Goal</Label>
        <Select id={`${id}-goal`} value={goalId} onChange={(e) => setGoalId(e.target.value)}>
          <option value="">No goal</option>
          {goals.map((g) => (
            <option key={g.id} value={g.id}>
              {g.title}
            </option>
          ))}
        </Select>
      </div>
      <AddField
        label="Project name"
        showLabel
        placeholder="Budgeting app"
        button="Add project"
        onAdd={(title) => actions.addProject(title, goalId || null)}
      />
    </section>
  );
}
