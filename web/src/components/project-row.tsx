import { Link } from "react-router";
import { nextTask, projectProgress, useAttachments } from "@/data/structure";
import type { Project, Task } from "@/data/types";
import { Progress } from "./goal-line";
import { AttachmentIcon, NotesIcon, ProjectIcon, TasksIcon } from "./icons";

/**
 * A project with its progress line, an overview of what it holds (tasks,
 * attachments, notes) and the next task. Approved: icons with counts.
 */
export function ProjectRow({ project, tasks }: { project: Project; tasks: Task[] }) {
  const p = projectProgress(project.id, tasks);
  const next = nextTask(project.id, tasks);
  const attachments = useAttachments({ kind: "project", id: project.id }).length;
  const hasNotes = project.notes.trim() !== "";
  return (
    <li className="border-t border-line py-3">
      <Link to={`/projects/${project.id}`} className="block min-h-11">
        <span className="flex items-center gap-2.5">
          <ProjectIcon size={20} className="shrink-0 text-muted-foreground" />
          <span className="min-w-0 flex-1 text-[17px]/[1.3] font-medium min-[900px]:text-lg/[1.3]">
            {project.title}
          </span>
          {project.status !== "open" && (
            <span className="text-[13px] text-muted-foreground">
              {project.status === "paused" ? "Paused" : "Done"}
            </span>
          )}
        </span>
        <Progress
          done={p.done}
          total={p.total}
          label={`${project.title}: ${p.done} of ${p.total} done`}
        />
        <span className="mt-1.5 flex flex-wrap items-center gap-x-4 gap-y-1 text-[13px] text-muted-foreground">
          <span className="inline-flex items-center gap-1.5">
            <TasksIcon size={16} />
            {p.total ? `${p.done} of ${p.total} done` : "No tasks yet"}
          </span>
          {attachments > 0 && (
            <span className="inline-flex items-center gap-1.5">
              <AttachmentIcon size={16} />
              {attachments}
              <span className="sr-only">{attachments === 1 ? " attachment" : " attachments"}</span>
            </span>
          )}
          {hasNotes && (
            <span className="inline-flex items-center gap-1.5">
              <NotesIcon size={16} />
              Notes
            </span>
          )}
        </span>
        <span className="mt-1 block text-[13px] text-muted-foreground">
          {next ? `Next: ${next.title}` : p.total ? "Nothing left to do" : ""}
        </span>
      </Link>
    </li>
  );
}
