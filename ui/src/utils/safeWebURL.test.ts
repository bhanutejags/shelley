import { strict as assert } from "node:assert";
import { safeWebURL } from "./safeWebURL";

assert.equal(
  safeWebURL("https://user:pass@example.org/path?token=secret#fragment"),
  "https://example.org/path",
);
assert.equal(
  safeWebURL("https://example.org/path?signature=secret#fragment"),
  "https://example.org/path",
);
assert.equal(safeWebURL("http://example.org/path"), "http://example.org/path");
assert.equal(safeWebURL("javascript:alert(1)"), "");
assert.equal(safeWebURL("not a URL"), "");
assert.equal(safeWebURL(""), "");

console.log("safeWebURL tests passed");
