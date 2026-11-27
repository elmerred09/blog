import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TopicList, TopicNav } from "@/components/topic-list";

describe("TopicList", () => {
  it("renders each tag, in order, as a link to its tag page", () => {
    render(
      <TopicList
        tags={[
          { slug: "go", name: "Go" },
          { slug: "postgres", name: "Postgres" },
        ]}
      />,
    );

    expect(
      screen.getByRole("heading", { name: "Dive in by topic" }),
    ).toBeTruthy();
    const links = within(screen.getByRole("list")).getAllByRole("link");
    expect(links.map((a) => a.textContent)).toEqual(["Go", "Postgres"]);
    expect(links.map((a) => a.getAttribute("href"))).toEqual([
      "/tags/go",
      "/tags/postgres",
    ]);
  });

  it("renders nothing when there are no tags", () => {
    render(<TopicList tags={[]} />);

    expect(screen.queryByRole("heading")).toBeNull();
    expect(screen.queryByRole("list")).toBeNull();
  });
});

describe("TopicNav", () => {
  const tags = [
    { slug: "go", name: "Go" },
    { slug: "postgres", name: "Postgres" },
  ];

  it("links every topic and marks the current one", () => {
    render(<TopicNav tags={tags} current="postgres" />);

    const nav = screen.getByRole("navigation", { name: "Topics" });
    const links = within(nav).getAllByRole("link");
    expect(links.map((a) => a.getAttribute("href"))).toEqual([
      "/tags/go",
      "/tags/postgres",
    ]);
    expect(links.map((a) => a.getAttribute("aria-current"))).toEqual([
      null,
      "page",
    ]);
  });

  it("renders nothing when there are no tags", () => {
    render(<TopicNav tags={[]} current="go" />);

    expect(screen.queryByRole("navigation")).toBeNull();
  });
});
