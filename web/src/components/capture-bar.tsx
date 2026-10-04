import { type FormEvent, useId, useState } from "react";
import { useActions } from "@/data/hooks";
import { cn } from "@/lib/utils";
import { Button } from "./ui/button";
import { Input } from "./ui/input";

// A text field and a square ink "Add" button. On the phone it sits right
// above the tab bar; captured tasks land in the Inbox.
export function CaptureBar({ fixed = false }: { fixed?: boolean }) {
  const actions = useActions();
  const [title, setTitle] = useState("");
  const [status, setStatus] = useState("");
  const id = useId();

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!title.trim()) return;
    await actions.addTask({ title });
    setTitle("");
    setStatus("Added to Inbox.");
  }

  return (
    <form
      onSubmit={submit}
      className={cn(
        "flex gap-2",
        fixed &&
          "fixed inset-x-0 bottom-[calc(72px+env(safe-area-inset-bottom))] z-10 border-t border-line bg-bg px-6 py-3",
      )}
    >
      <label htmlFor={id} className="sr-only">
        New task
      </label>
      <Input
        id={id}
        value={title}
        onChange={(e) => {
          setTitle(e.target.value);
          setStatus("");
        }}
        placeholder="Capture a task"
        maxLength={300}
        autoComplete="off"
        enterKeyHint="done"
      />
      <Button
        type="submit"
        variant="primary"
        className="h-12 w-14 shrink-0 px-0 text-sm"
        aria-label="Add task"
      >
        Add
      </Button>
      <p className="sr-only" aria-live="polite">
        {status}
      </p>
    </form>
  );
}
