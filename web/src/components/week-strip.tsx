import { useEffect, useMemo, useState } from "react";
import { useTasks, useToday, useWeekStart } from "@/data/hooks";
import type { LocalDate } from "@/data/types";
import {
  addDays,
  dayOfMonth,
  longDate,
  startOfWeek,
  weekdayLetter,
  weekOf,
  weekRange,
} from "@/lib/dates";
import { cn } from "@/lib/utils";
import { BackIcon, NextIcon } from "./icons";

interface WeekStripProps {
  selected: LocalDate | null;
  /** Called with the tapped day, or null when the selected day is tapped again. */
  onSelect: (day: LocalDate | null) => void;
}

// Seven equal cells: weekday letter, date, up to three dots for how much is
// planned or due. Approved: arrows move a week at a time, and the range label
// returns to this week. Weeks start on settings.weekStart.
export function WeekStrip({ selected, onSelect }: WeekStripProps) {
  const today = useToday();
  const weekStart = useWeekStart();
  const tasks = useTasks();
  const thisWeek = startOfWeek(today, weekStart);
  const [first, setFirst] = useState(() => startOfWeek(selected ?? today, weekStart));

  // A day chosen elsewhere (say, from a link) brings its week into view.
  useEffect(() => {
    if (selected) setFirst(startOfWeek(selected, weekStart));
  }, [selected, weekStart]);

  const days = weekOf(first, weekStart);
  const counts = useMemo(() => {
    const m = new Map<string, number>();
    for (const t of tasks) {
      if (t.status === "dropped") continue;
      const seen = new Set<string>();
      if (t.plannedOn) seen.add(t.plannedOn);
      if (t.due && t.status === "open") seen.add(t.due);
      for (const d of seen) m.set(d, (m.get(d) ?? 0) + 1);
    }
    return m;
  }, [tasks]);

  const range = weekRange(first);
  const away = first !== thisWeek;

  return (
    <div>
      <div className="mb-1 flex items-center justify-between">
        <button
          type="button"
          aria-label="Previous week"
          onClick={() => setFirst(addDays(first, -7))}
          className="grid size-11 place-items-center rounded-md"
        >
          <BackIcon size={20} />
        </button>
        {away ? (
          <button
            type="button"
            onClick={() => setFirst(thisWeek)}
            aria-label={`${range}. Back to this week`}
            className="min-h-11 rounded-md px-2 text-sm font-semibold underline underline-offset-[3px]"
          >
            {range}
          </button>
        ) : (
          <p className="text-sm font-semibold" aria-live="polite">
            {range}
          </p>
        )}
        <button
          type="button"
          aria-label="Next week"
          onClick={() => setFirst(addDays(first, 7))}
          className="grid size-11 place-items-center rounded-md"
        >
          <NextIcon size={20} />
        </button>
      </div>
      <ul className="grid grid-cols-7 gap-1" aria-label={`Week of ${range}`}>
        {days.map((d) => {
          const n = counts.get(d) ?? 0;
          const label = `${longDate(d)}${d === today ? ", today" : ""}, ${n === 0 ? "nothing" : n === 1 ? "1 task" : `${n} tasks`}`;
          return (
            <li key={d}>
              <button
                type="button"
                className={cn(
                  "flex min-h-[68px] w-full flex-col items-center justify-center gap-1 rounded-md",
                  d === selected && "bg-soft-day",
                )}
                aria-pressed={d === selected}
                aria-label={label}
                onClick={() => onSelect(d === selected ? null : d)}
              >
                <span className="text-xs font-medium text-muted-foreground">
                  {weekdayLetter(d)}
                </span>
                <span
                  className={cn(
                    "text-[17px] font-semibold",
                    d === today && "underline underline-offset-4",
                  )}
                >
                  {dayOfMonth(d)}
                </span>
                <span className="flex h-1.5 gap-[3px]" aria-hidden="true">
                  {Array.from({ length: Math.min(n, 3) }, (_, i) => (
                    // biome-ignore lint/suspicious/noArrayIndexKey: dots are identical
                    <span key={i} className="size-1.5 rounded-full bg-ink" />
                  ))}
                </span>
              </button>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
