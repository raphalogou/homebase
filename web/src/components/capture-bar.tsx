import { type FormEvent, useEffect, useId, useRef, useState } from "react";
import { useActions } from "@/data/hooks";
import { cn } from "@/lib/utils";
import { InboxIcon } from "./icons";
import { useNotify } from "./toaster";
import { Button } from "./ui/button";
import { Input } from "./ui/input";

// A text field and a square ink "Add" button. On the phone it sits right
// above the tab bar; captured tasks land in the Inbox.
export function CaptureBar({ fixed = false }: { fixed?: boolean }) {
  const actions = useActions();
  const [title, setTitle] = useState("");
  const notify = useNotify();
  const id = useId();
  const form = useRef<HTMLFormElement>(null);

  // On the phone the bar sits where toasts appear; lift them above it.
  useEffect(() => {
    const el = form.current;
    if (!fixed || !el) return;
    const root = document.documentElement;
    const lift = () => root.style.setProperty("--toast-lift", `${el.offsetHeight}px`);
    lift();
    const ro = new ResizeObserver(lift);
    ro.observe(el);
    return () => {
      ro.disconnect();
      root.style.removeProperty("--toast-lift");
    };
  }, [fixed]);

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!title.trim()) return;
    const value = title;
    setTitle("");
    await actions.addTask({ title: value });
    notify({ title: `Added to Inbox: ${value.trim()}`, icon: InboxIcon });
  }

  return (
    <form
      ref={form}
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
        }}
        placeholder="Capture a task"
        data-capture
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
    </form>
  );
}
