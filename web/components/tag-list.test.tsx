import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TagList } from "@/components/tag-list";

describe("TagList", () => {
  it("renders each tag, in order, as a link home", () => {
    render(
      <TagList
        tags={[
          { slug: "postgres", name: "Postgres" },
          { slug: "go", name: "Go" },
        ]}
      />,
    );

    const list = screen.getByRole("list", { name: "Tags" });
    const links = within(list).getAllByRole("link");
    expect(links.map((a) => a.textContent)).toEqual(["Postgres", "Go"]);
    expect(links.map((a) => a.getAttribute("href"))).toEqual(["/", "/"]);
  });

  it("renders nothing when there are no tags", () => {
    render(<TagList tags={[]} />);

    expect(screen.queryByRole("list", { name: "Tags" })).toBeNull();
  });
});
