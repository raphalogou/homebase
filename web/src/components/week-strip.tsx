import { useMemo } from "react";
import { useTasks, useToday, useWeekStart } from "@/data/hooks";
import type { LocalDate } from "@/data/types";
import { dayOfMonth, longDate, weekdayLetter, weekOf } from "@/lib/dates";
import { cn } from "@/lib/utils";

interface WeekStripProps {
  selected: LocalDate | null;
  onSelect?: (day: LocalDate | null) => void;
}

// Seven equal cells: weekday letter, date, up to three dots for how much is
// planned or due. Weeks start on settings.weekStart.
export function WeekStrip({ selected, onSelect }: WeekStripProps) {
  const today = useToday();
  const weekStart = useWeekStart();
  const tasks = useTasks();
  const days = weekOf(today, weekStart);

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

  return (
    <ul className="grid grid-cols-7 gap-1" aria-label="This week">
      {days.map((d) => {
        const n = counts.get(d) ?? 0;
        const label = `${longDate(d)}, ${n === 0 ? "nothing" : n === 1 ? "1 task" : `${n} tasks`}`;
        const cell = (
          <>
            <span className="text-xs font-medium text-muted-foreground">{weekdayLetter(d)}</span>
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
          </>
        );
        const cls = cn(
          "flex min-h-[68px] w-full flex-col items-center justify-center gap-1 rounded-md",
          d === selected && "bg-soft-day",
        );
        return (
          <li key={d}>
            {onSelect ? (
              <button
                type="button"
                className={cls}
                aria-pressed={d === selected}
                aria-label={label}
                onClick={() => onSelect(d === selected ? null : d)}
              >
                {cell}
              </button>
            ) : (
              <div className={cls} role="img" aria-label={label}>
                {cell}
              </div>
            )}
          </li>
        );
      })}
    </ul>
  );
}
