import { describe, expect, it } from "vitest";

import { sameDraft, type Draft } from "./draft";

function draft(overrides: Partial<Draft> = {}): Draft {
  return {
    title: "Keyset pagination",
    slug: "keyset-pagination",
    summary: "Why OFFSET gets slow.",
    body: "Hello.",
    tags: ["Postgres"],
    publish: { mode: "draft", scheduleLocal: "" },
    ...overrides,
  };
}

describe("sameDraft", () => {
  it("matches an unchanged draft", () => {
    expect(sameDraft(draft(), draft())).toBe(true);
  });

  it.each([
    ["title", { title: "Renamed" }],
    ["slug", { slug: "renamed" }],
    ["summary", { summary: "A different summary." }],
    ["body", { body: "Changed." }],
    ["tags", { tags: ["Postgres", "Go"] }],
    ["tag order", { tags: ["Go", "Postgres"] }],
    ["publish mode", { publish: { mode: "now" as const, scheduleLocal: "" } }],
  ])("differs when %s changes", (_label, overrides) => {
    expect(sameDraft(draft(), draft(overrides))).toBe(false);
  });

  it("ignores a leftover schedule time outside schedule mode", () => {
    expect(
      sameDraft(
        draft(),
        draft({
          publish: { mode: "draft", scheduleLocal: "2026-10-20T09:30" },
        }),
      ),
    ).toBe(true);
  });

  it("differs when the scheduled time changes", () => {
    const scheduled = draft({
      publish: { mode: "schedule", scheduleLocal: "2026-10-20T09:30" },
    });
    expect(
      sameDraft(
        scheduled,
        draft({
          publish: { mode: "schedule", scheduleLocal: "2026-10-21T09:30" },
        }),
      ),
    ).toBe(false);
  });
});
