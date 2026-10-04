import assert from "node:assert/strict";
import { test } from "node:test";
import { type IndicatorInput, indicatorFor } from "./sync-indicator.ts";

const calm: IndicatorInput = {
  state: "idle",
  slow: false,
  trouble: null,
  recovered: false,
  waiting: 0,
  retryIn: 0,
};

test("indicatorFor shows nothing when synced, and syncing only when slow", () => {
  assert.deepEqual(indicatorFor(calm), { kind: "none" });
  assert.deepEqual(indicatorFor({ ...calm, state: "syncing" }), { kind: "none" });
  assert.deepEqual(indicatorFor({ ...calm, state: "syncing", slow: true }), { kind: "syncing" });
});

test("indicatorFor keeps trouble showing through quick retries", () => {
  assert.deepEqual(indicatorFor({ ...calm, state: "offline", waiting: 2 }), {
    kind: "offline",
    waiting: 2,
  });
  assert.deepEqual(indicatorFor({ ...calm, state: "syncing", trouble: "offline", waiting: 2 }), {
    kind: "offline",
    waiting: 2,
  });
  assert.deepEqual(indicatorFor({ ...calm, state: "syncing", trouble: "offline", slow: true }), {
    kind: "syncing",
  });
  assert.deepEqual(indicatorFor({ ...calm, state: "error", retryIn: 30 }), {
    kind: "error",
    retryIn: 30,
  });
});

test("indicatorFor says synced after trouble clears", () => {
  assert.deepEqual(indicatorFor({ ...calm, recovered: true }), { kind: "recovered" });
});
