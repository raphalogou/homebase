import { useParams } from "react-router";
import { AddField } from "@/components/add-field";
import { Columns } from "@/components/app-shell";
import { Attachments } from "@/components/attachments";
import { BackLink } from "@/components/back-link";
import { Progress } from "@/components/goal-line";
import { NotesField } from "@/components/notes-field";
import { SectionLabel } from "@/components/section";
import { DetailSkeleton, ErrorBlock } from "@/components/states";
import { StatusAndDelete } from "@/components/status-and-delete";
import { TaskRow } from "@/components/task-row";
import { TitleField } from "@/components/title-field";
import { useActions, useTasks } from "@/data/hooks";
import {
  projectProgress,
  useGoal,
  useProject,
  useProjectTasks,
  useStructureActions,
} from "@/data/structure";
import { useFirstLoad } from "@/data/sync-status";

export default function ProjectScreen() {
  const { id } = useParams();
  const project = useProject(id);
  const goal = useGoal(project?.goalId ?? undefined);
  const tasks = useTasks();
  const list = useProjectTasks(id ?? "");
  const actions = useActions();
  const structure = useStructureActions();
  const { loading } = useFirstLoad();

  if (!project) {
    return (
      <Columns
        main={
          <>
            <BackLink to="/plan/projects">Plan</BackLink>
            <div className="mt-6">
              {loading ? (
                <DetailSkeleton what="project" />
              ) : (
                <ErrorBlock
                  title="This project no longer exists"
                  action={{ label: "Back to Plan", to: "/plan/projects" }}
                >
                  It may have been deleted on another device.
                </ErrorBlock>
              )}
            </div>
          </>
        }
      />
    );
  }

  const p = projectProgress(project.id, tasks);
  const main = (
    <>
      {goal ? (
        <BackLink to={`/goals/${goal.id}`}>{goal.title}</BackLink>
      ) : (
        <BackLink to="/plan/projects">Projects</BackLink>
      )}
      <h1 className="mt-3">
        <TitleField
          label="Project name"
          value={project.title}
          onSave={(title) => void structure.updateProject(project.id, { title })}
          className="text-[30px]/[1.15] font-bold min-[900px]:text-[32px]/[1.15]"
        />
      </h1>
      <Progress done={p.done} total={p.total} label={`${p.done} of ${p.total} tasks done`} />
      <p className="mt-1.5 text-[13px] text-muted-foreground">
        {p.total ? `${p.done} of ${p.total} tasks done` : "No tasks yet"}
      </p>

      <section aria-labelledby="project-tasks" className="mt-10">
        <SectionLabel id="project-tasks">Tasks</SectionLabel>
        {list.length === 0 && (
          <p className="mb-3 border-t border-line pt-3 text-muted-foreground">
            No tasks yet. Add the first one.
          </p>
        )}
        <AddField
          label="New task in this project"
          placeholder="Add a task"
          button="Add task"
          onAdd={(title) => actions.addTask({ title, projectId: project.id })}
        />
        <ul className="mt-3">
          {list.map((t) => (
            <li key={t.id}>
              <TaskRow task={t} hideContext onToggle={() => void actions.toggleDone(t.id)} />
            </li>
          ))}
        </ul>
      </section>

      <div className="mt-10">
        <NotesField
          value={project.notes}
          onSave={(notes) => structure.updateProject(project.id, { notes })}
        />
      </div>

      <div className="mt-10">
        <Attachments owner={{ kind: "project", id: project.id }} />
      </div>

      <StatusAndDelete
        kind="project"
        id={project.id}
        title={project.title}
        status={project.status}
        after={goal ? `/goals/${goal.id}` : "/plan/projects"}
      />
    </>
  );
  return <Columns main={main} />;
}
