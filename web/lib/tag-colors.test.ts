import { describe, expect, it } from "vitest";

import { tagChipClasses, tagColors } from "@/lib/tag-colors";

describe("tagColors", () => {
  it("lists every palette color once, slate first", () => {
    expect(tagColors[0]).toBe("slate");
    expect(new Set(tagColors).size).toBe(tagColors.length);
    expect(tagColors).toHaveLength(16);
  });
});

describe("tagChipClasses", () => {
  it("gives each color its own tint and text", () => {
    for (const color of tagColors) {
      expect(tagChipClasses(color)).toBe(
        `bg-tag-${color} text-tag-${color}-text`,
      );
    }
  });

  it("falls back to slate for a color this build doesn't know", () => {
    expect(tagChipClasses("magenta" as never)).toBe(tagChipClasses("slate"));
  });
});
