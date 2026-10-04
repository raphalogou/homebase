import assert from "node:assert/strict";
import { test } from "node:test";
import { normaliseUrl } from "./urls.ts";

test("normaliseUrl", () => {
  const cases: [string, string | null][] = [
    ["example.com", "https://example.com/"],
    ["  https://example.com/plan?x=1  ", "https://example.com/plan?x=1"],
    ["http://intranet.local/a", "http://intranet.local/a"],
    ["mailto:coach@example.com", "mailto:coach@example.com"],
    ["javascript:alert(1)", null],
    ["data:text/html,hi", null],
    ["ftp://example.com", null],
    ["localhost", null],
    ["", null],
  ];
  for (const [input, want] of cases) {
    assert.equal(normaliseUrl(input), want, input);
  }
});
