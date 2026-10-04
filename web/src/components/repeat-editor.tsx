import { useEffect, useId, useState } from "react";
import { useActions, useRepeats, useWeekStart } from "@/data/hooks";
import type { Repeat, Task } from "@/data/types";
import { longDate } from "@/lib/dates";
import { describeRepeat, unitWord, weekdaysFrom } from "@/lib/repeat";
import { cn } from "@/lib/utils";
import { RepeatIcon } from "./icons";
import { Button } from "./ui/button";
import { Input, Label, Select } from "./ui/input";
import { Segmented } from "./ui/segmented";

type Rule = Pick<Repeat, "freq" | "every" | "mode" | "weekdays" | "until">;

// Approved design: the repeat fields grow in place in the task form and
// save as they change, like the rest of the form.
export function RepeatEditor({
  task,
  run,
}: {
  task: Task;
  run: (fn: () => Promise<unknown>) => Promise<void>;
}) {
  const actions = useActions();
  const repeats = useRepeats();
  const weekStart = useWeekStart();
  const id = useId();
  const existing = task.repeatId ? repeats.get(task.repeatId) : undefined;
  const rule: Rule | null =
    existing && existing.deletedAt === null
      ? {
          freq: existing.freq,
          every: existing.every,
          mode: existing.mode,
          weekdays: existing.weekdays,
          until: existing.until,
        }
      : null;
  const [every, setEvery] = useState(String(rule?.every ?? 1));
  useEffect(() => setEvery(String(rule?.every ?? 1)), [rule?.every]);

  const save = (patch: Partial<Rule>) => {
    if (!rule) return;
    const next = { ...rule, ...patch };
    // Weekdays mean something only for weekly rules counted from the due date.
    if (next.freq !== "week" || next.mode !== "fixed") next.weekdays = null;
    void run(() => actions.setRepeat(task.id, next));
  };

  if (!rule) {
    return (
      <div>
        <span className="mb-2 block text-sm font-semibold text-muted-foreground">Repeat</span>
        <div className="flex items-center justify-between gap-3">
          <span>Does not repeat</span>
          <Button
            variant="small"
            onClick={() =>
              void run(() =>
                actions.setRepeat(task.id, { freq: "week", every: 1, mode: "after_done" }),
              )
            }
          >
            <RepeatIcon size={18} />
            Make it repeat
          </Button>
        </div>
      </div>
    );
  }

  const days = weekdaysFrom(weekStart);
  return (
    <fieldset className="min-w-0">
      <legend className="mb-2 text-sm font-semibold text-muted-foreground">Repeat</legend>

      <div className="flex gap-2">
        <div className="w-24">
          <Label htmlFor={`${id}-every`} className="sr-only">
            Every how many
          </Label>
          <Input
            id={`${id}-every`}
            type="number"
            inputMode="numeric"
            min={1}
            max={365}
            value={every}
            onChange={(e) => setEvery(e.target.value)}
            onBlur={() => {
              const n = Math.round(Number(every));
              if (Number.isFinite(n) && n >= 1 && n <= 365) {
                if (n !== rule.every) save({ every: n });
              } else {
                setEvery(String(rule.every));
              }
            }}
          />
        </div>
        <div className="flex-1">
          <Label htmlFor={`${id}-unit`} className="sr-only">
            Unit
          </Label>
          <Select
            id={`${id}-unit`}
            value={rule.freq}
            onChange={(e) => save({ freq: e.target.value as Rule["freq"] })}
          >
            {(["day", "week", "month"] as const).map((f) => (
              <option key={f} value={f}>
                {unitWord(f, Number(every) || 1)}
              </option>
            ))}
          </Select>
        </div>
      </div>

      <div className="mt-4">
        <span className="mb-2 block text-sm font-semibold text-muted-foreground">Counts from</span>
        <Segmented<Rule["mode"]>
          label="Counts from"
          value={rule.mode}
          options={[
            { value: "after_done", label: "When it's done" },
            { value: "fixed", label: "The due date" },
          ]}
          onChange={(mode) => save({ mode })}
        />
      </div>

      {rule.freq === "week" && rule.mode === "fixed" && (
        <div className="mt-4">
          <fieldset className="min-w-0">
            <legend className="mb-2 text-sm font-semibold text-muted-foreground">On</legend>
            <div className="flex gap-1">
              {days.map((d) => {
                const on = ((rule.weekdays ?? 0) & d.bit) !== 0;
                return (
                  <button
                    key={d.bit}
                    type="button"
                    aria-pressed={on}
                    aria-label={d.name}
                    onClick={() => {
                      const mask = (rule.weekdays ?? 0) ^ d.bit;
                      save({ weekdays: mask === 0 ? null : mask });
                    }}
                    className={cn(
                      "grid size-11 place-items-center rounded-full border text-sm font-semibold",
                      on ? "border-ink bg-ink text-field" : "border-line bg-field",
                    )}
                  >
                    {d.letter}
                  </button>
                );
              })}
            </div>
          </fieldset>
        </div>
      )}

      <div className="mt-4">
        <span className="mb-2 block text-sm font-semibold text-muted-foreground">Ends</span>
        <Segmented<"never" | "date">
          label="Ends"
          value={rule.until ? "date" : "never"}
          options={[
            { value: "never", label: "Never" },
            { value: "date", label: "On a date" },
          ]}
          onChange={(v) => {
            if (v === "never") save({ until: null });
            else if (!rule.until) {
              const d = new Date();
              d.setMonth(d.getMonth() + 3);
              save({ until: d.toISOString().slice(0, 10) });
            }
          }}
        />
        {rule.until && (
          <div className="mt-2">
            <Label htmlFor={`${id}-until`} className="sr-only">
              Last day
            </Label>
            <Input
              id={`${id}-until`}
              type="date"
              value={rule.until}
              onChange={(e) => e.target.value && save({ until: e.target.value })}
            />
          </div>
        )}
      </div>

      <p className="mt-4 font-serif text-[17px]/[1.35]">{describeRepeat(rule, longDate)}</p>
      <p className="mt-1 text-[13px] text-muted-foreground">
        The next one is added when you finish this one.
      </p>
      <Button
        variant="text"
        className="mt-2"
        onClick={() => void run(() => actions.setRepeat(task.id, null))}
      >
        Stop repeating
      </Button>
    </fieldset>
  );
}
