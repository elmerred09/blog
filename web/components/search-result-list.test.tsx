import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { SearchResultList } from "@/components/search-result-list";
import type { SearchResult } from "@/lib/api";

const results: SearchResult[] = [
  {
    slug: "keyset-pagination",
    title: "Keyset pagination",
    summary: "Why OFFSET gets slow.",
    published_at: "2026-10-02T12:00:00Z",
    tags: [{ slug: "postgres", name: "Postgres" }],
    snippet: [
      { text: "use ", match: false },
      { text: "keyset", match: true },
    ],
  },
  {
    slug: "hello-world",
    title: "Hello, world",
    summary: "The first post.",
    published_at: "2026-10-01T12:00:00Z",
    tags: [],
    snippet: [],
  },
];

describe("SearchResultList", () => {
  it("renders one entry per result, in the order given", () => {
    render(<SearchResultList query="keyset" results={results} />);

    const items = screen.getAllByRole("article");
    expect(
      items.map((a) => within(a).getByRole("heading").textContent),
    ).toEqual(["Keyset pagination", "Hello, world"]);
  });

  it("links each title to the post page", () => {
    render(<SearchResultList query="keyset" results={results} />);

    const link = screen.getByRole("link", { name: "Keyset pagination" });
    expect(link.getAttribute("href")).toBe("/posts/keyset-pagination");
  });

  it("shows the snippet with its matches, and the date and tags", () => {
    render(<SearchResultList query="keyset" results={results} />);

    const first = screen.getAllByRole("article")[0];
    expect(first.querySelector("mark")?.textContent).toBe("keyset");
    expect(within(first).getByText("Oct 2, 2026")).toBeTruthy();
    expect(within(first).getByText("Postgres")).toBeTruthy();
  });

  it("falls back to the summary when there's no snippet", () => {
    render(<SearchResultList query="keyset" results={results} />);

    const second = screen.getAllByRole("article")[1];
    expect(within(second).getByText("The first post.")).toBeTruthy();
  });

  it("says nothing matched, quoting the query", () => {
    render(<SearchResultList query="no such thing" results={[]} />);

    expect(screen.queryByRole("article")).toBeNull();
    expect(screen.getByText("No posts match “no such thing”.")).toBeTruthy();
  });
});
