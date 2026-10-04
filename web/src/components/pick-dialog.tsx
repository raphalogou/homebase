import { Dialog } from "@base-ui/react/dialog";
import { useState } from "react";
import {
  ActionError,
  MAX_PER_DAY,
  useActions,
  usePlanned,
  useSuggestions,
  useToday,
} from "@/data/hooks";
import type { Task } from "@/data/types";
import { useOpenPick } from "@/lib/open-pick";
import { useIsDesktop } from "@/lib/use-media";
import { cn } from "@/lib/utils";
import { CheckIcon, NoticeIcon, PlannedIcon } from "./icons";
import { Empty, SectionLabel } from "./section";
import { useTaskMeta } from "./task-row";
import { useNotify } from "./toaster";
import { Button } from "./ui/button";
import { Sheet } from "./ui/sheet";

const TITLE = "Choose up to three";

// "Pickers and edit forms open as a bottom sheet" (DESIGN.md); on desktop the
// same picker is a centred dialog.
export function PickDialog() {
  const [open, setOpen] = useOpenPick();
  const desktop = useIsDesktop();
  if (!desktop) {
    return (
      <Sheet open={open} onOpenChange={setOpen} title={TITLE} showTitle>
        <PickBody onDone={() => setOpen(false)} />
      </Sheet>
    );
  }
  return (
    <Dialog.Root open={open} onOpenChange={(o) => setOpen(o)}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-40 bg-ink/30" />
        <Dialog.Popup className="fixed top-1/2 left-1/2 z-50 flex max-h-[85dvh] w-[min(560px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 flex-col rounded-[20px] bg-bg px-8 pt-8">
          <Dialog.Title className="mb-1 text-[26px]/[1.2] font-bold">{TITLE}</Dialog.Title>
          <div className="-mx-8 overflow-y-auto px-8">
            <PickBody onDone={() => setOpen(false)} />
          </div>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

function PickBody({ onDone }: { onDone: () => void }) {
  const today = useToday();
  const planned = usePlanned(today);
  const s = useSuggestions();
  const actions = useActions();
  const notify = useNotify();

  const done = planned.filter((t) => t.status === "done");
  const current = planned.filter((t) => t.status === "open");
  const [chosen, setChosen] = useState<string[]>(() => current.map((t) => t.id));
  const max = MAX_PER_DAY - done.length;

  const groups: [string, Task[]][] = [
    ["Planned for today", current],
    ["Due soon", s.dueSoon],
    ["In progress", s.inProgress],
    ["Moved a few times", s.moved],
  ];
  const any = groups.some(([, list]) => list.length > 0);

  function toggle(id: string) {
    setChosen((c) =>
      c.includes(id) ? c.filter((x) => x !== id) : c.length < max ? [...c, id] : c,
    );
  }

  async function save() {
    try {
      await actions.setToday(chosen);
      notify({
        title: chosen.length ? "Today is set" : "Nothing planned for today",
        icon: PlannedIcon,
      });
      onDone();
    } catch (err) {
      if (err instanceof ActionError) notify({ title: err.message, icon: NoticeIcon });
      else throw err;
    }
  }

  return (
    <div>
      <p className="text-muted-foreground">
        {done.length > 0
          ? `${done.length} done already, so ${max === 1 ? "one more fits" : `${max} more fit`}.`
          : "Pick what matters most today. Untick one to take it off."}
      </p>
      {!any && (
        <div className="mt-6">
          <Empty>Nothing to choose from yet. Capture a task first.</Empty>
        </div>
      )}
      {groups.map(([label, list]) =>
        list.length === 0 ? null : (
          <section key={label} className="mt-6" aria-label={label}>
            <SectionLabel>{label}</SectionLabel>
            <ul>
              {list.map((t) => (
                <PickRow
                  key={t.id}
                  task={t}
                  checked={chosen.includes(t.id)}
                  disabled={!chosen.includes(t.id) && chosen.length >= max}
                  onToggle={() => toggle(t.id)}
                />
              ))}
            </ul>
          </section>
        ),
      )}
      <div className="sticky bottom-0 -mx-8 mt-6 border-t border-line bg-bg px-8 py-4 max-[899px]:-mx-6 max-[899px]:px-6">
        <Button className="w-full" onClick={() => void save()}>
          Set today ({chosen.length} chosen)
        </Button>
      </div>
    </div>
  );
}

function PickRow({
  task,
  checked,
  disabled,
  onToggle,
}: {
  task: Task;
  checked: boolean;
  disabled: boolean;
  onToggle: () => void;
}) {
  const meta = useTaskMeta()(task);
  return (
    <li className="border-t border-line">
      <label
        className={cn(
          "flex min-h-16 w-full cursor-pointer items-center gap-3 py-2.5",
          disabled && "cursor-default opacity-50",
        )}
      >
        <input
          type="checkbox"
          checked={checked}
          disabled={disabled}
          onChange={onToggle}
          className="peer sr-only"
        />
        <span
          aria-hidden="true"
          className={cn(
            "grid size-[26px] shrink-0 place-items-center rounded-md border-[1.8px] border-ink peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-ink",
            checked && "bg-ink text-field",
          )}
        >
          {checked && <CheckIcon size={18} />}
        </span>
        <span className="flex min-w-0 flex-col">
          <span className="text-[17px]/[1.3] font-medium">{task.title}</span>
          {meta && <span className="mt-0.5 text-[13px] text-muted-foreground">{meta}</span>}
        </span>
      </label>
    </li>
  );
}
