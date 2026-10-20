import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

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
    published_at: "2026-10-01T12:00:00Z",
    updated_at: "2026-10-02T08:30:00Z",
    state: "published",
  },
];

describe("AdminPostList", () => {
  it("links each title to its editor", () => {
    render(<AdminPostList posts={items} />);

    expect(
      screen.getByRole("link", { name: "A draft" }).getAttribute("href"),
    ).toBe(`/admin/posts/${items[0].id}`);
  });

  it("shows each post's state and updated time", () => {
    render(<AdminPostList posts={items} />);

    const rows = screen.getAllByRole("row").slice(1);
    expect(
      rows.map((row) => within(row).getByTestId("state").textContent),
    ).toEqual(["Draft", "Scheduled", "Published"]);
    expect(
      within(rows[0]).getByText("Oct 18, 2026, 4:05 PM UTC"),
    ).toBeDefined();
  });

  it("shows the publish date only for posts that have one", () => {
    render(<AdminPostList posts={items} />);

    const published = screen
      .getAllByRole("row")
      .slice(1)
      .map((row) => within(row).getAllByRole("cell")[2].textContent);
    expect(published).toEqual([
      "—",
      "Oct 25, 2026, 9:00 AM UTC",
      "Oct 1, 2026, 12:00 PM UTC",
    ]);
  });

  it("says so when there are no posts", () => {
    render(<AdminPostList posts={[]} />);

    expect(screen.queryByRole("table")).toBeNull();
    expect(screen.getByText(/No posts yet/)).toBeDefined();
  });
});
