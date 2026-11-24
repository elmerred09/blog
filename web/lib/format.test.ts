import { describe, expect, it, vi } from "vitest";

import {
  formatCompactDateTime,
  formatDate,
  formatDateTime,
  formatReadingTime,
  formatZonedDateTime,
} from "@/lib/format";

describe("formatReadingTime", () => {
  it.each([
    { minutes: 1, want: "1 min read" },
    { minutes: 10, want: "10 min read" },
  ])("formats $minutes as $want", ({ minutes, want }) => {
    expect(formatReadingTime(minutes)).toBe(want);
  });
});

describe("formatDateTime", () => {
  it("formats in UTC and says so", () => {
    vi.stubEnv("TZ", "Pacific/Auckland");

    expect(formatDateTime("2026-10-18T16:05:59.123456Z")).toBe(
      "Oct 18, 2026, 4:05 PM UTC",
    );
  });
});

describe("formatCompactDateTime", () => {
  const now = new Date("2026-11-24T17:00:00Z");

  it.each([
    {
      name: "drops the year within the current one",
      iso: "2026-10-18T16:05:00Z",
      timeZone: "UTC",
      want: "Oct 18, 4:05 PM",
    },
    {
      name: "shows the time in the given zone",
      iso: "2026-10-18T16:05:00Z",
      timeZone: "America/New_York",
      want: "Oct 18, 12:05 PM",
    },
    {
      name: "shows only the date outside the current year",
      iso: "2025-03-02T09:00:00Z",
      timeZone: "UTC",
      want: "Mar 2, 2025",
    },
    {
      name: "compares years in the given zone",
      iso: "2027-01-01T03:00:00Z",
      timeZone: "America/Los_Angeles",
      want: "Dec 31, 7:00 PM",
    },
  ])("$name", ({ iso, timeZone, want }) => {
    expect(formatCompactDateTime(iso, { now, timeZone })).toBe(want);
  });
});

describe("formatZonedDateTime", () => {
  it("gives the full date and time with the zone's name", () => {
    expect(
      formatZonedDateTime("2026-10-18T16:05:00Z", {
        timeZone: "America/New_York",
      }),
    ).toBe("Oct 18, 2026, 12:05 PM EDT");
  });
});

describe("formatDate", () => {
  it.each([
    { iso: "2026-10-01T12:00:00Z", want: "Oct 1, 2026" },
    { iso: "2026-01-31T00:00:00Z", want: "Jan 31, 2026" },
    { iso: "2026-12-31T23:59:59Z", want: "Dec 31, 2026" },
  ])("formats $iso as $want", ({ iso, want }) => {
    expect(formatDate(iso)).toBe(want);
  });

  // The server and browser can be in different time zones; the date must be
  // the UTC one regardless.
  it.each(["America/Los_Angeles", "Pacific/Auckland"])(
    "uses UTC when the process is in %s",
    (tz) => {
      vi.stubEnv("TZ", tz);
      expect(formatDate("2026-10-01T02:00:00Z")).toBe("Oct 1, 2026");
      expect(formatDate("2026-10-01T22:00:00Z")).toBe("Oct 1, 2026");
    },
  );
});
