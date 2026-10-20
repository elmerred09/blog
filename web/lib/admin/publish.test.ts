import { beforeEach, describe, expect, it, vi } from "vitest";

import { initialPublishState, publishedAtFor, toLocalInput } from "./publish";

const now = new Date("2026-10-18T16:00:00Z");

beforeEach(() => {
  // UTC-4 in October, so local and UTC visibly differ.
  vi.stubEnv("TZ", "America/New_York");
});

describe("initialPublishState", () => {
  it("starts a post without published_at as a draft", () => {
    expect(initialPublishState(null, now)).toEqual({
      mode: "draft",
      scheduleLocal: "",
    });
  });

  it("starts a post published in the past as published", () => {
    expect(initialPublishState("2026-10-01T12:00:00Z", now).mode).toBe("now");
  });

  it("shows a scheduled post's time in local time", () => {
    expect(initialPublishState("2026-10-20T13:30:00Z", now)).toEqual({
      mode: "schedule",
      scheduleLocal: "2026-10-20T09:30",
    });
  });
});

describe("publishedAtFor", () => {
  it("sends null for a draft", () => {
    expect(
      publishedAtFor({ mode: "draft", scheduleLocal: "" }, null, now),
    ).toEqual({ ok: true, value: null });
  });

  it("publishes a new post now", () => {
    expect(
      publishedAtFor({ mode: "now", scheduleLocal: "" }, null, now),
    ).toEqual({ ok: true, value: "2026-10-18T16:00:00.000Z" });
  });

  it("keeps a published post's original date", () => {
    const original = "2026-10-01T12:00:00.123456Z";

    expect(
      publishedAtFor({ mode: "now", scheduleLocal: "" }, original, now),
    ).toEqual({ ok: true, value: original });
  });

  it("publishes a scheduled post now rather than at its scheduled time", () => {
    expect(
      publishedAtFor(
        { mode: "now", scheduleLocal: "" },
        "2026-10-20T13:30:00Z",
        now,
      ),
    ).toEqual({ ok: true, value: "2026-10-18T16:00:00.000Z" });
  });

  it("sends a scheduled local time as UTC", () => {
    expect(
      publishedAtFor(
        { mode: "schedule", scheduleLocal: "2026-10-20T09:30" },
        null,
        now,
      ),
    ).toEqual({ ok: true, value: "2026-10-20T13:30:00.000Z" });
  });

  it.each([
    ["no time", ""],
    ["a past time", "2026-10-18T11:59"],
    ["an unparseable time", "tomorrow"],
  ])("rejects a schedule with %s", (_, scheduleLocal) => {
    const result = publishedAtFor(
      { mode: "schedule", scheduleLocal },
      null,
      now,
    );

    expect(result.ok).toBe(false);
  });
});

describe("toLocalInput", () => {
  it("formats an instant for a datetime-local input", () => {
    expect(toLocalInput("2026-01-05T03:07:00Z")).toBe("2026-01-04T22:07");
  });
});
