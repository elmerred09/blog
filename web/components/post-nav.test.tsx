import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PostNav } from "@/components/post-nav";

const newer = { slug: "newer-post", title: "Newer post" };
const older = { slug: "older-post", title: "Older post" };

describe("PostNav", () => {
  it("links the older post first and the newer post second", () => {
    render(<PostNav adjacent={{ newer, older }} />);

    const nav = screen.getByRole("navigation", { name: "More posts" });
    const links = within(nav).getAllByRole("link");
    expect(links.map((link) => link.getAttribute("href"))).toEqual([
      "/posts/older-post",
      "/posts/newer-post",
    ]);
    expect(
      within(nav).getByRole("link", { name: "Older bit: Older post" }),
    ).toBe(links[0]);
    expect(
      within(nav).getByRole("link", { name: "Newer bit: Newer post" }),
    ).toBe(links[1]);
  });

  it("keeps the newer post on the right when there is no older one", () => {
    render(<PostNav adjacent={{ newer, older: null }} />);

    const link = screen.getByRole("link");
    expect(link.getAttribute("href")).toBe("/posts/newer-post");
    expect(link.className).toContain("sm:col-start-2");
  });

  it("renders nothing when the post has no neighbors", () => {
    const { container } = render(
      <PostNav adjacent={{ newer: null, older: null }} />,
    );

    expect(container.childElementCount).toBe(0);
  });
});
