import { render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AdminPostList } from "@/components/admin/admin-post-list";
import type { AdminPostListItem } from "@/lib/api/types";

const items: AdminPostListItem[] = [
  {
    id: "0f8a5c1e-0000-4000-8000-000000000001",
    slug: "draft-post",
    title: "A draft",
    published_at: null,
    updated_at: "2026-10-18T16:05:00.123456Z",
    state: "draft",
  },
  {
    id: "0f8a5c1e-0000-4000-8000-000000000002",
    slug: "scheduled-post",
    title: "Coming soon",
    published_at: "2026-10-25T09:00:00Z",
    updated_at: "2026-10-17T10:00:00Z",
    state: "scheduled",
  },
  {
    id: "0f8a5c1e-0000-4000-8000-000000000003",
    slug: "live-post",
    title: "Live",
    published_at: "2025-10-01T12:00:00Z",
    updated_at: "2025-10-02T08:30:00Z",
    state: "published",
  },
];

function bodyRows() {
  return screen.getAllByRole("row").slice(1);
}

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: vi.fn(), refresh: vi.fn() }),
}));
vi.mock("@/app/admin/actions", () => ({ deletePost: vi.fn() }));

describe("AdminPostList", () => {
  beforeEach(() => {
    vi.stubEnv("TZ", "America/New_York");
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(new Date("2026-11-24T17:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("links each title and pencil to its editor", () => {
    render(<AdminPostList posts={items} />);

    const href = `/admin/posts/${items[0].id}`;
    expect(
      screen.getByRole("link", { name: "A draft" }).getAttribute("href"),
    ).toBe(href);
    expect(
      screen.getByRole("link", { name: "Edit A draft" }).getAttribute("href"),
    ).toBe(href);
  });

  it("has a delete button per post", () => {
    render(<AdminPostList posts={items} />);

    expect(
      screen
        .getByRole("button", { name: "Delete A draft" })
        .getAttribute("aria-disabled"),
    ).toBeNull();
  });

  it("shows each post's state", () => {
    render(<AdminPostList posts={items} />);

    expect(
      bodyRows().map((row) => within(row).getByTestId("state").textContent),
    ).toEqual(["Draft", "Scheduled", "Published"]);
  });

  it("shows times in the browser's zone, without the year when it's this one", () => {
    render(<AdminPostList posts={items} />);

    const cells = (column: number) =>
      bodyRows().map(
        (row) => within(row).getAllByRole("cell")[column].textContent,
      );
    expect(cells(2)).toEqual(["—", "Oct 25, 5:00 AM", "Oct 1, 2025"]);
    expect(cells(3)).toEqual([
      "Oct 18, 12:05 PM",
      "Oct 17, 6:00 AM",
      "Oct 2, 2025",
    ]);
  });

  it("gives the full time and zone on hover", () => {
    render(<AdminPostList posts={items} />);

    const updated = within(bodyRows()[0]).getByText("Oct 18, 12:05 PM");
    expect(updated.getAttribute("dateTime")).toBe(items[0].updated_at);
    expect(updated.getAttribute("title")).toBe("Oct 18, 2026, 12:05 PM EDT");
  });

  it("says so when there are no posts", () => {
    render(<AdminPostList posts={[]} />);

    expect(screen.queryByRole("table")).toBeNull();
    expect(screen.getByText(/No posts yet/)).toBeDefined();
  });
});
