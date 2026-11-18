import { describe, expect, it } from "vitest";

import { titleSize } from "@/lib/og";

describe("titleSize", () => {
  it.each([
    ["a short title at full size", "a".repeat(40), 76],
    ["a medium title smaller", "a".repeat(41), 64],
    ["a long title smallest", "a".repeat(73), 52],
  ])("sets %s", (_, title, size) => {
    expect(titleSize(title)).toBe(size);
  });
});
