import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TagList } from "@/components/tag-list";

describe("TagList", () => {
  it("renders each tag, in order, as a link to its tag page", () => {
    render(
      <TagList
        tags={[
          { slug: "postgres", name: "Postgres", color: "slate" },
          { slug: "go", name: "Go", color: "slate" },
        ]}
      />,
    );

    const list = screen.getByRole("list", { name: "Tags" });
    const links = within(list).getAllByRole("link");
    expect(links.map((a) => a.textContent)).toEqual(["Postgres", "Go"]);
    expect(links.map((a) => a.getAttribute("href"))).toEqual([
      "/tags/postgres",
      "/tags/go",
    ]);
  });

  it("colors each chip by its tag's palette color", () => {
    render(
      <TagList
        tags={[
          { slug: "postgres", name: "Postgres", color: "teal" },
          { slug: "go", name: "Go", color: "slate" },
        ]}
      />,
    );

    const [postgres, go] = within(
      screen.getByRole("list", { name: "Tags" }),
    ).getAllByRole("link");
    expect(postgres.className).toContain("bg-tag-teal");
    expect(postgres.className).toContain("text-tag-teal-text");
    expect(go.className).toContain("bg-tag-slate");
  });

  it("renders nothing when there are no tags", () => {
    render(<TagList tags={[]} />);

    expect(screen.queryByRole("list", { name: "Tags" })).toBeNull();
  });
});
