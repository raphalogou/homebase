import type { ReactNode } from "react";
import { useContextOf, useToday } from "@/data/hooks";
import type { Task } from "@/data/types";
import { dueText } from "@/lib/dates";
import { useOpenTask } from "@/lib/open-task";
import { cn } from "@/lib/utils";
import { DoneBox } from "./checkbox";

/** Project or goal, due text, repeat note: the line under a task's title. */
export function useTaskMeta(): (t: Task) => string {
  const contextOf = useContextOf();
  const today = useToday();
  return (t: Task) => {
    const parts: string[] = [];
    const why = contextOf(t);
    if (why) parts.push(why);
    if (t.due && t.status === "open") parts.push(dueText(t.due, today));
    if (t.repeatId) parts.push("Repeats");
    return parts.join(" · ");
  };
}

export function TaskTitle({ task, today }: { task: Task; today: string }) {
  const done = task.status === "done";
  const marked = task.plannedOn === today && task.status === "open";
  return (
    <span
      className={cn(
        "text-[17px]/[1.3] font-medium min-[900px]:text-lg/[1.3]",
        done && "text-muted-foreground line-through",
        task.status === "dropped" && "text-muted-foreground",
      )}
    >
      <span className={cn(marked && "mark")}>{task.title}</span>
    </span>
  );
}

interface TaskRowProps {
  task: Task;
  onToggle: () => void;
  /** Before the checkbox: a rank number or a grip handle. */
  lead?: ReactNode;
  /** After the title: buttons such as "Today". */
  trail?: ReactNode;
  className?: string;
}

// DESIGN.md "Task row": at least 64 px, a hairline above, no box. Tapping the
// title opens the task; the checkbox toggles done.
export function TaskRow({ task, onToggle, lead, trail, className }: TaskRowProps) {
  const today = useToday();
  const meta = useTaskMeta()(task);
  const [, open] = useOpenTask();
  return (
    <div className={cn("flex min-h-16 items-center gap-3 border-t border-line py-2.5", className)}>
      {lead}
      <DoneBox done={task.status === "done"} onToggle={onToggle} title={task.title} />
      <button
        type="button"
        onClick={() => open(task.id)}
        className="flex min-h-11 min-w-0 flex-1 flex-col items-start justify-center text-left"
      >
        <TaskTitle task={task} today={today} />
        {meta && <span className="mt-0.5 text-[13px] text-muted-foreground">{meta}</span>}
      </button>
      {trail}
    </div>
  );
}
