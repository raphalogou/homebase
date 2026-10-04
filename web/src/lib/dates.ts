// Local-day arithmetic on "YYYY-MM-DD" strings. A date string is turned into
// UTC midnight only to do arithmetic and to name weekdays, and is formatted
// back in UTC, so no time zone can shift the day.

import type { LocalDate } from "../data/types.ts";

const DAY_MS = 86_400_000;

/** The calendar day of instant `now` in time zone `tz` (settings.tz). */
export function todayIn(tz: string, now: number = Date.now()): LocalDate {
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: tz,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(now);
  const get = (type: string) => parts.find((p) => p.type === type)?.value ?? "";
  return `${get("year")}-${get("month")}-${get("day")}`;
}

function toUTC(date: LocalDate): number {
  const [y, m, d] = date.split("-").map(Number);
  return Date.UTC(y ?? 1970, (m ?? 1) - 1, d ?? 1);
}

function fromUTC(ms: number): LocalDate {
  return new Date(ms).toISOString().slice(0, 10);
}

export function isDate(s: string): boolean {
  return /^\d{4}-\d{2}-\d{2}$/.test(s) && fromUTC(toUTC(s)) === s;
}

export function addDays(date: LocalDate, n: number): LocalDate {
  return fromUTC(toUTC(date) + n * DAY_MS);
}

/** Days from a to b; positive when b is later. */
export function daysBetween(a: LocalDate, b: LocalDate): number {
  return Math.round((toUTC(b) - toUTC(a)) / DAY_MS);
}

/** 0 for Sunday to 6 for Saturday. */
export function weekday(date: LocalDate): number {
  return new Date(toUTC(date)).getUTCDay();
}

/** The first day of date's week; weekStart is 1 for Monday, 0 for Sunday. */
export function startOfWeek(date: LocalDate, weekStart: 0 | 1): LocalDate {
  const back = (weekday(date) - weekStart + 7) % 7;
  return addDays(date, -back);
}

export function weekOf(date: LocalDate, weekStart: 0 | 1): LocalDate[] {
  const first = startOfWeek(date, weekStart);
  return Array.from({ length: 7 }, (_, i) => addDays(first, i));
}

function fmt(date: LocalDate, options: Intl.DateTimeFormatOptions): string {
  return new Intl.DateTimeFormat("en-GB", { ...options, timeZone: "UTC" }).format(toUTC(date));
}

/** "Wednesday 4 March" */
export function longDate(date: LocalDate): string {
  return fmt(date, { weekday: "long", day: "numeric", month: "long" });
}

/** "W" */
export function weekdayLetter(date: LocalDate): string {
  return fmt(date, { weekday: "narrow" });
}

export function dayOfMonth(date: LocalDate): number {
  return new Date(toUTC(date)).getUTCDate();
}

/** Short text for a due date, as in DESIGN.md: "Today", "Tomorrow", "Friday", "2d overdue". */
export function dueText(due: LocalDate, today: LocalDate): string {
  const d = daysBetween(today, due);
  if (d < 0) return `${-d}d overdue`;
  if (d === 0) return "Today";
  if (d === 1) return "Tomorrow";
  if (d < 7) return fmt(due, { weekday: "long" });
  const sameYear = due.slice(0, 4) === today.slice(0, 4);
  return fmt(
    due,
    sameYear
      ? { day: "numeric", month: "short" }
      : { day: "numeric", month: "short", year: "numeric" },
  );
}

/** Heading for a group of tasks on one day: "Today", "Tomorrow", "Friday 6 March". */
export function dayHeading(date: LocalDate, today: LocalDate): string {
  const d = daysBetween(today, date);
  if (d === 0) return "Today";
  if (d === 1) return "Tomorrow";
  if (d === -1) return "Yesterday";
  return fmt(date, { weekday: "long", day: "numeric", month: "long" });
}

/** "6 – 12 October", or "28 September – 4 October" across months. */
export function weekRange(first: LocalDate): string {
  const last = addDays(first, 6);
  const sameMonth = first.slice(0, 7) === last.slice(0, 7);
  const from = fmt(first, sameMonth ? { day: "numeric" } : { day: "numeric", month: "long" });
  return `${from} – ${fmt(last, { day: "numeric", month: "long" })}`;
}
