import assert from "node:assert/strict";
import { test } from "node:test";
import { describeRepeat, unitWord, weekdaysFrom } from "./repeat.ts";

const fmt = (d: string) => d;

test("describeRepeat", () => {
  const cases: [Parameters<typeof describeRepeat>[0], string][] = [
    [
      { freq: "day", every: 1, mode: "after_done", weekdays: null, until: null },
      "Every day, counted from when it is done.",
    ],
    [
      { freq: "week", every: 2, mode: "fixed", weekdays: 1 | 8, until: null },
      "Every 2 weeks on Monday and Thursday, from the due date.",
    ],
    [
      { freq: "week", every: 1, mode: "fixed", weekdays: 1 | 4 | 16, until: null },
      "Every week on Monday, Wednesday and Friday, from the due date.",
    ],
    // Weekdays only apply when counting from the due date.
    [
      { freq: "week", every: 1, mode: "after_done", weekdays: 1, until: null },
      "Every week, counted from when it is done.",
    ],
    [
      { freq: "month", every: 3, mode: "fixed", weekdays: null, until: "2026-12-31" },
      "Every 3 months, from the due date. Until 2026-12-31.",
    ],
  ];
  for (const [rule, want] of cases) {
    assert.equal(describeRepeat(rule, fmt), want);
  }
});

test("units and week order", () => {
  assert.equal(unitWord("week", 1), "week");
  assert.equal(unitWord("week", 2), "weeks");
  assert.equal(weekdaysFrom(1)[0]?.name, "Monday");
  assert.equal(weekdaysFrom(0)[0]?.name, "Sunday");
  assert.equal(weekdaysFrom(0).length, 7);
});
