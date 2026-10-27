import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TopicList } from "@/components/topic-list";

describe("TopicList", () => {
  it("renders each tag, in order, as a link home", () => {
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
    expect(links.map((a) => a.getAttribute("href"))).toEqual(["/", "/"]);
  });

  it("renders nothing when there are no tags", () => {
    render(<TopicList tags={[]} />);

    expect(screen.queryByRole("heading")).toBeNull();
    expect(screen.queryByRole("list")).toBeNull();
  });
});
