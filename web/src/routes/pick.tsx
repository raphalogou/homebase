import { useState } from "react";
import { Link, useNavigate } from "react-router";
import { Columns } from "@/components/app-shell";
import { CheckIcon } from "@/components/icons";
import { Empty, ScreenTitle, SectionLabel } from "@/components/section";
import { useTaskMeta } from "@/components/task-row";
import { Button } from "@/components/ui/button";
import {
  ActionError,
  MAX_PER_DAY,
  useActions,
  usePlanned,
  useSuggestions,
  useToday,
} from "@/data/hooks";
import type { Task } from "@/data/types";
import { cn } from "@/lib/utils";

// Phone only in the design; on desktop the same choices sit beside Today.
export default function Pick() {
  const today = useToday();
  const planned = usePlanned(today);
  const s = useSuggestions();
  const actions = useActions();
  const navigate = useNavigate();
  const [message, setMessage] = useState("");

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
      navigate("/");
    } catch (err) {
      if (err instanceof ActionError) setMessage(err.message);
      else throw err;
    }
  }

  const main = (
    <>
      <Link
        to="/"
        className="inline-flex min-h-11 items-center font-semibold underline underline-offset-[3px]"
      >
        Today
      </Link>
      <ScreenTitle className="mt-2">Choose up to three</ScreenTitle>
      <p className="mt-2 text-muted-foreground">
        {done.length > 0
          ? `${done.length} done already, so ${max === 1 ? "one more fits" : `${max} more fit`}.`
          : "Pick what matters most today."}
      </p>

      {!any && (
        <div className="mt-8">
          <Empty>Nothing to choose from yet. Capture a task first.</Empty>
        </div>
      )}
      {groups.map(([label, list]) =>
        list.length === 0 ? null : (
          <section key={label} className="mt-8" aria-label={label}>
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

      <div className="fixed inset-x-0 bottom-[calc(72px+env(safe-area-inset-bottom))] z-10 border-t border-line bg-bg px-6 py-3 min-[900px]:static min-[900px]:mt-10 min-[900px]:border-0 min-[900px]:p-0">
        {message && (
          <p className="mb-2 text-sm" role="status">
            {message}
          </p>
        )}
        <Button className="w-full min-[900px]:w-auto" onClick={() => void save()}>
          Set today ({chosen.length} chosen)
        </Button>
      </div>
    </>
  );

  return <Columns main={main} capture />;
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
