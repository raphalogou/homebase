// Words for a repeat rule, and the weekday mask of the repeats table
// (Mon=1, Tue=2, Wed=4, ... Sun=64).

import type { Repeat } from "../data/types.ts";

export const WEEKDAYS = [
  { bit: 1, name: "Monday", letter: "M" },
  { bit: 2, name: "Tuesday", letter: "T" },
  { bit: 4, name: "Wednesday", letter: "W" },
  { bit: 8, name: "Thursday", letter: "T" },
  { bit: 16, name: "Friday", letter: "F" },
  { bit: 32, name: "Saturday", letter: "S" },
  { bit: 64, name: "Sunday", letter: "S" },
] as const;

/** The days in display order for the week start (1 Monday, 0 Sunday). */
export function weekdaysFrom(weekStart: 0 | 1) {
  return weekStart === 1 ? [...WEEKDAYS] : [WEEKDAYS[6], ...WEEKDAYS.slice(0, 6)];
}

const UNIT: Record<Repeat["freq"], [string, string]> = {
  day: ["day", "days"],
  week: ["week", "weeks"],
  month: ["month", "months"],
};

export function unitWord(freq: Repeat["freq"], every: number): string {
  return UNIT[freq][every === 1 ? 0 : 1];
}

function list(words: string[]): string {
  if (words.length <= 1) return words.join("");
  return `${words.slice(0, -1).join(", ")} and ${words[words.length - 1]}`;
}

/** "Every 2 weeks on Monday and Thursday, from the due date. Until 30 June." */
export function describeRepeat(
  r: Pick<Repeat, "freq" | "every" | "mode" | "weekdays" | "until">,
  formatDate: (d: string) => string,
): string {
  let s = r.every === 1 ? `Every ${UNIT[r.freq][0]}` : `Every ${r.every} ${UNIT[r.freq][1]}`;
  if (r.freq === "week" && r.mode === "fixed" && r.weekdays) {
    s += ` on ${list(WEEKDAYS.filter((d) => (r.weekdays ?? 0) & d.bit).map((d) => d.name))}`;
  }
  s += r.mode === "fixed" ? ", from the due date." : ", counted from when it is done.";
  if (r.until) s += ` Until ${formatDate(r.until)}.`;
  return s;
}
