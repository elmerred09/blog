import { describe, expect, it } from "vitest";

import { slugify } from "./slug";

// The cases from internal/slug's tests, so the suggestion matches what the
// API would make from the same title.
describe("slugify", () => {
  it.each([
    ["Postgres", "postgres"],
    ["Postgres Internals", "postgres-internals"],
    ["  Go  ", "go"],
    ["Node.js", "node-js"],
    ["snake_case", "snake-case"],
    ["Go 1.27", "go-1-27"],
    ["full-text search", "full-text-search"],
    ["a  --  b", "a-b"],
    ["--Edge Cases!--", "edge-cases"],
    ["Café Über", "cafe-uber"],
    ["keyset-pagination", "keyset-pagination"],
  ])("turns %j into %j", (title, want) => {
    expect(slugify(title)).toBe(want);
  });

  it.each(["", "   ", "!!!", "日本語"])(
    "returns an empty slug for %j",
    (title) => {
      expect(slugify(title)).toBe("");
    },
  );
});
