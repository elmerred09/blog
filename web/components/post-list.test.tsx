import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PostList } from "@/components/post-list";
import type { PostListItem } from "@/lib/api";

const posts: PostListItem[] = [
  {
    slug: "keyset-pagination",
    title: "Keyset pagination",
    summary: "Why OFFSET gets slow.",
    published_at: "2026-10-02T12:00:00Z",
    tags: [{ slug: "postgres", name: "Postgres" }],
  },
  {
    slug: "hello-world",
    title: "Hello, world",
    summary: "The first post.",
    published_at: "2026-10-01T12:00:00Z",
    tags: [],
  },
];

describe("PostList", () => {
  it("renders one entry per post, in order", () => {
    render(<PostList posts={posts} />);

    const items = screen.getAllByRole("article");
    expect(items).toHaveLength(2);
    expect(within(items[0]).getByRole("heading").textContent).toBe(
      "Keyset pagination",
    );
    expect(within(items[1]).getByRole("heading").textContent).toBe(
      "Hello, world",
    );
  });

  it("links each title to the post page", () => {
    render(<PostList posts={posts} />);

    const link = screen.getByRole("link", { name: "Keyset pagination" });
    expect(link.getAttribute("href")).toBe("/posts/keyset-pagination");
  });

  it("shows the summary, date and tags", () => {
    render(<PostList posts={posts} />);

    const first = screen.getAllByRole("article")[0];
    expect(within(first).getByText("Why OFFSET gets slow.")).toBeTruthy();
    expect(within(first).getByText("Oct 2, 2026")).toBeTruthy();
    expect(within(first).getByText("Postgres")).toBeTruthy();
  });

  it("says so when there are no posts", () => {
    render(<PostList posts={[]} />);

    expect(screen.queryByRole("article")).toBeNull();
    expect(screen.getByText("No posts yet.")).toBeTruthy();
  });
});
