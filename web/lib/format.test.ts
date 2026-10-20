import { describe, expect, it, vi } from "vitest";

import { formatDate, formatDateTime } from "@/lib/format";

describe("formatDateTime", () => {
  it("formats in UTC and says so", () => {
    vi.stubEnv("TZ", "Pacific/Auckland");

    expect(formatDateTime("2026-10-18T16:05:59.123456Z")).toBe(
      "Oct 18, 2026, 4:05 PM UTC",
    );
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
