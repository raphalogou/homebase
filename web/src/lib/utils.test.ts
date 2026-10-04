import assert from "node:assert/strict";
import { test } from "node:test";
import { cn } from "./utils.ts";

test("cn lets a later Tailwind class override an earlier one", () => {
  assert.equal(cn("px-2 text-ink", "px-4"), "text-ink px-4");
});

test("cn drops falsy values", () => {
  const done = false;
  assert.equal(cn("row", done && "line-through", undefined), "row");
});
