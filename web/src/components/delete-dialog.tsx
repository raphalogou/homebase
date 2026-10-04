import { Button } from "./ui/button";
import { ResponsiveDialog } from "./ui/dialog";

interface DeleteDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  /** What is inside, such as "It has 4 tasks and 2 attachments." Empty when nothing is. */
  contents: string;
  keepLabel: string;
  deleteAllLabel: string;
  /** Used when there is nothing to keep: one plain delete button. */
  deleteLabel: string;
  onKeep: () => void;
  onDeleteAll: () => void;
}

// Approved design: a bottom sheet on the phone, a small centred dialog on
// desktop. Keeping the tasks is the first, primary choice.
export function DeleteDialog(props: DeleteDialogProps) {
  return (
    <ResponsiveDialog open={props.open} onOpenChange={props.onOpenChange} title={props.title}>
      <DeleteChoices {...props} />
    </ResponsiveDialog>
  );
}

function DeleteChoices({
  contents,
  keepLabel,
  deleteAllLabel,
  deleteLabel,
  onKeep,
  onDeleteAll,
  onOpenChange,
}: DeleteDialogProps) {
  return (
    <div className="flex flex-col gap-3 pb-2">
      {contents && <p className="mb-3 text-muted-foreground">{contents}</p>}
      {contents ? (
        <>
          <Button onClick={onKeep}>{keepLabel}</Button>
          <Button variant="secondary" onClick={onDeleteAll}>
            {deleteAllLabel}
          </Button>
        </>
      ) : (
        <Button onClick={onDeleteAll}>{deleteLabel}</Button>
      )}
      <Button variant="text" className="self-center" onClick={() => onOpenChange(false)}>
        Cancel
      </Button>
    </div>
  );
}

/** "It has 2 projects, 4 tasks and 1 attachment." */
export function contentsText(c: { projects: number; tasks: number; attachments: number }): string {
  const parts: string[] = [];
  const plural = (n: number, one: string) => `${n} ${one}${n === 1 ? "" : "s"}`;
  if (c.projects) parts.push(plural(c.projects, "project"));
  if (c.tasks) parts.push(plural(c.tasks, "task"));
  if (c.attachments) parts.push(plural(c.attachments, "attachment"));
  if (parts.length === 0) return "";
  const last = parts.pop();
  return `It has ${parts.length ? `${parts.join(", ")} and ${last}` : last}.`;
}
