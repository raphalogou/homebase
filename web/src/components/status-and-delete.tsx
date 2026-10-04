import { useId, useState } from "react";
import { useNavigate } from "react-router";
import { useContents, useStructureActions } from "@/data/structure";
import type { GoalStatus } from "@/data/types";
import { contentsText, DeleteDialog } from "./delete-dialog";
import { DeleteIcon } from "./icons";
import { useNotify } from "./toaster";
import { Button } from "./ui/button";
import { Label, Select } from "./ui/input";

const STATUS: { value: GoalStatus; label: string }[] = [
  { value: "open", label: "Open" },
  { value: "paused", label: "Paused" },
  { value: "done", label: "Done" },
  { value: "dropped", label: "Dropped" },
];

/** The foot of a goal or project: its status, and deleting it after asking. */
export function StatusAndDelete({
  kind,
  id,
  title,
  status,
  after,
}: {
  kind: "goal" | "project";
  id: string;
  title: string;
  status: GoalStatus;
  /** Where to go once it is deleted. */
  after: string;
}) {
  const actions = useStructureActions();
  const contents = useContents();
  const navigate = useNavigate();
  const notify = useNotify();
  const [open, setOpen] = useState(false);
  const fieldId = useId();
  const c = kind === "goal" ? contents.ofGoal(id) : contents.ofProject(id);
  const noun = kind === "goal" ? "goal" : "project";

  async function remove(withContents: boolean) {
    setOpen(false);
    navigate(after, { replace: true });
    const undo =
      kind === "goal"
        ? await actions.deleteGoal(id, withContents)
        : await actions.deleteProject(id, withContents);
    notify({
      title: `Deleted “${title}”`,
      icon: DeleteIcon,
      action: {
        label: "Undo",
        run: () =>
          void undo().then(() => navigate(kind === "goal" ? `/goals/${id}` : `/projects/${id}`)),
      },
    });
  }

  return (
    <div className="mt-10 flex flex-wrap items-end justify-between gap-4 border-t border-line pt-6">
      <div className="w-48">
        <Label htmlFor={fieldId}>Status</Label>
        <Select
          id={fieldId}
          value={status}
          onChange={(e) => void actions.setStatus(kind, id, e.target.value as GoalStatus)}
        >
          {STATUS.map((s) => (
            <option key={s.value} value={s.value}>
              {s.label}
            </option>
          ))}
        </Select>
      </div>
      <Button variant="text" onClick={() => setOpen(true)}>
        Delete {noun}
      </Button>
      <DeleteDialog
        open={open}
        onOpenChange={setOpen}
        title={`Delete “${title}”?`}
        contents={contentsText(c)}
        keepLabel={kind === "goal" ? "Keep its projects and tasks" : "Keep tasks, delete project"}
        deleteAllLabel={
          kind === "goal" ? "Delete goal and everything in it" : "Delete project and tasks"
        }
        deleteLabel={`Delete ${noun}`}
        onKeep={() => void remove(false)}
        onDeleteAll={() => void remove(true)}
      />
    </div>
  );
}
