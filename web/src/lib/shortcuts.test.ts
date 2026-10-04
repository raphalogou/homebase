import assert from "node:assert/strict";
import { test } from "node:test";
import { type KeyInput, shortcutFor } from "./shortcuts.ts";

const press = (key: string, extra: Partial<KeyInput> = {}): KeyInput => ({
  key,
  ctrlKey: false,
  altKey: false,
  metaKey: false,
  typing: false,
  ...extra,
});

test("shortcutFor maps the single keys", () => {
  assert.deepEqual(shortcutFor(press("c")), { kind: "capture" });
  assert.deepEqual(shortcutFor(press("T")), { kind: "go", to: "/" });
  assert.deepEqual(shortcutFor(press("i")), { kind: "go", to: "/inbox" });
  assert.deepEqual(shortcutFor(press("p")), { kind: "go", to: "/plan" });
  assert.deepEqual(shortcutFor(press("g")), { kind: "go", to: "/goals" });
  assert.deepEqual(shortcutFor(press("r")), { kind: "go", to: "/review" });
  assert.deepEqual(shortcutFor(press("s")), { kind: "modal", name: "settings" });
  assert.deepEqual(shortcutFor(press("?")), { kind: "help" });
  assert.equal(shortcutFor(press("x")), null);
  assert.equal(shortcutFor(press("Escape")), null);
});

test("shortcutFor stays out of the way while typing or with modifiers", () => {
  assert.equal(shortcutFor(press("t", { typing: true })), null);
  assert.equal(shortcutFor(press("c", { ctrlKey: true })), null);
  assert.equal(shortcutFor(press("p", { metaKey: true })), null);
  assert.equal(shortcutFor(press("g", { altKey: true })), null);
});
