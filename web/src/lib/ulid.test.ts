import assert from "node:assert/strict";
import { test } from "node:test";
import { ulid } from "./ulid.ts";

test("ulid is 26 Crockford characters and sorts by time", () => {
  const a = ulid(1_700_000_000_000);
  const b = ulid(1_700_000_000_001);
  assert.match(a, /^[0-7][0-9A-HJKMNP-TV-Z]{25}$/);
  assert.ok(a.slice(0, 10) < b.slice(0, 10));
});

test("ulid encodes the time like the server", () => {
  // The first 10 characters are the 48-bit millisecond time.
  assert.equal(ulid(0).slice(0, 10), "0000000000");
  assert.equal(ulid(2 ** 48 - 1).slice(0, 10), "7ZZZZZZZZZ");
});

test("ulid does not repeat", () => {
  const seen = new Set(Array.from({ length: 1000 }, () => ulid(5)));
  assert.equal(seen.size, 1000);
});
