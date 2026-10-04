import assert from "node:assert/strict";
import { test } from "node:test";
import {
  confirmError,
  normalizeUsername,
  passphraseError,
  USERNAME_ERROR,
  usernameError,
} from "./account.ts";

test("usernameError follows the server's rule", () => {
  const cases: [string, boolean][] = [
    ["sam", true],
    [" Sam.Lee_2-x ", true],
    ["ab", false],
    ["a".repeat(32), true],
    ["a".repeat(33), false],
    ["sam lee", false],
    ["sam@home", false],
    ["sàm", false],
  ];
  for (const [input, ok] of cases) {
    assert.equal(usernameError(input), ok ? "" : USERNAME_ERROR, input);
  }
  assert.equal(normalizeUsername("  Sam "), "sam");
});

test("passphraseError counts characters and says how many", () => {
  assert.equal(passphraseError("river lamp"), "Use at least 12 characters. You have 10.");
  assert.equal(passphraseError("river lamp o"), "");
  assert.equal(passphraseError("été été été"), "Use at least 12 characters. You have 11.");
  assert.equal(passphraseError("🌲".repeat(12)), "");
  assert.equal(passphraseError("a".repeat(1001)), "Use at most 1000 characters.");
});

test("confirmError compares exactly", () => {
  assert.equal(confirmError("maple window", "maple window"), "");
  assert.equal(confirmError("maple window", "Maple window"), "The two passphrases do not match.");
});
