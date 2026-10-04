import assert from "node:assert/strict";
import { test } from "node:test";
import {
  addDays,
  dayHeading,
  daysBetween,
  dueText,
  isDate,
  longDate,
  startOfWeek,
  todayIn,
  weekday,
  weekOf,
  weekRange,
} from "./dates.ts";

test("todayIn uses the given zone, not the machine's", () => {
  // 23:30 UTC on 4 March 2026.
  const now = Date.UTC(2026, 2, 4, 23, 30);
  const cases: [string, string][] = [
    ["UTC", "2026-03-04"],
    ["Asia/Tokyo", "2026-03-05"],
    ["America/Los_Angeles", "2026-03-04"],
  ];
  for (const [tz, want] of cases) {
    assert.equal(todayIn(tz, now), want, tz);
  }
});

test("addDays crosses months, years and leap days", () => {
  const cases: [string, number, string][] = [
    ["2026-02-28", 1, "2026-03-01"],
    ["2028-02-28", 1, "2028-02-29"],
    ["2026-12-31", 1, "2027-01-01"],
    ["2026-03-01", -1, "2026-02-28"],
    // Daylight saving starts on 29 March 2026 in Europe; days are still days.
    ["2026-03-28", 2, "2026-03-30"],
  ];
  for (const [from, n, want] of cases) {
    assert.equal(addDays(from, n), want, `${from} + ${n}`);
  }
});

test("daysBetween and weekday", () => {
  assert.equal(daysBetween("2026-03-04", "2026-03-06"), 2);
  assert.equal(daysBetween("2026-03-06", "2026-03-04"), -2);
  assert.equal(weekday("2026-03-04"), 3); // Wednesday
});

test("startOfWeek honours the week start setting", () => {
  assert.equal(startOfWeek("2026-03-04", 1), "2026-03-02");
  assert.equal(startOfWeek("2026-03-04", 0), "2026-03-01");
  assert.equal(startOfWeek("2026-03-08", 1), "2026-03-02"); // Sunday in a Monday week
  assert.equal(startOfWeek("2026-03-02", 1), "2026-03-02");
  assert.deepEqual(weekOf("2026-03-04", 1), [
    "2026-03-02",
    "2026-03-03",
    "2026-03-04",
    "2026-03-05",
    "2026-03-06",
    "2026-03-07",
    "2026-03-08",
  ]);
});

test("dueText", () => {
  const today = "2026-03-04";
  const cases: [string, string][] = [
    ["2026-03-02", "2d overdue"],
    ["2026-03-04", "Today"],
    ["2026-03-05", "Tomorrow"],
    ["2026-03-06", "Friday"],
    ["2026-03-20", "20 Mar"],
    ["2027-01-10", "10 Jan 2027"],
  ];
  for (const [due, want] of cases) {
    assert.equal(dueText(due, today), want, due);
  }
});

test("headings and long dates", () => {
  assert.equal(longDate("2026-03-04"), "Wednesday 4 March");
  assert.equal(dayHeading("2026-03-04", "2026-03-04"), "Today");
  assert.equal(dayHeading("2026-03-05", "2026-03-04"), "Tomorrow");
  assert.equal(dayHeading("2026-03-06", "2026-03-04"), "Friday 6 March");
});

test("isDate", () => {
  assert.equal(isDate("2026-02-28"), true);
  assert.equal(isDate("2026-02-30"), false);
  assert.equal(isDate("2026-2-28"), false);
});

test("weekRange", () => {
  assert.equal(weekRange("2026-10-05"), "5 – 11 October");
  assert.equal(weekRange("2026-09-28"), "28 September – 4 October");
});
