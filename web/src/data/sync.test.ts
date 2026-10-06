import assert from "node:assert/strict";
import { test } from "node:test";
import { nextPoll, POLL_MAX_MS, POLL_MIN_MS } from "./sync.ts";

test("the timed check slows while nothing changes and resets on a change", () => {
  let delay = POLL_MIN_MS;
  const quiet: number[] = [];
  for (let i = 0; i < 6; i++) {
    delay = nextPoll(delay, false);
    quiet.push(delay / 60_000);
  }
  assert.deepEqual(quiet, [2, 4, 8, 10, 10, 10]);
  assert.equal(nextPoll(POLL_MAX_MS, true), POLL_MIN_MS);
});
