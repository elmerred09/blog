import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PostMeta } from "@/components/post-meta";

describe("PostMeta", () => {
  it("renders the date in a time element with the ISO timestamp", () => {
    render(<PostMeta publishedAt="2026-10-01T12:00:00Z" tags={[]} />);

    const time = screen.getByText("Oct 1, 2026");
    expect(time.tagName).toBe("TIME");
    expect(time.getAttribute("datetime")).toBe("2026-10-01T12:00:00Z");
  });

  it("renders tags in the order given", () => {
    render(
      <PostMeta
        publishedAt="2026-10-01T12:00:00Z"
        tags={[
          { slug: "postgres", name: "Postgres" },
          { slug: "go", name: "Go" },
          { slug: "api-design", name: "API design" },
        ]}
      />,
    );

    const list = screen.getByRole("list", { name: "Tags" });
    const names = within(list)
      .getAllByRole("listitem")
      .map((li) => li.textContent);
    expect(names).toEqual(["Postgres", "Go", "API design"]);
  });

  it("omits the tag list when there are no tags", () => {
    render(<PostMeta publishedAt="2026-10-01T12:00:00Z" tags={[]} />);

    expect(screen.queryByRole("list", { name: "Tags" })).toBeNull();
  });
});
