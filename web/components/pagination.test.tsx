import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Pagination } from "@/components/pagination";

describe("Pagination", () => {
  it("links to the newer and older pages", () => {
    render(<Pagination prevCursor="newer" nextCursor="older" />);

    expect(
      screen.getByRole("link", { name: "Newer posts" }).getAttribute("href"),
    ).toBe("/?cursor=newer");
    expect(
      screen.getByRole("link", { name: "Older posts" }).getAttribute("href"),
    ).toBe("/?cursor=older");
  });

  it("links only forward on the first page", () => {
    render(<Pagination prevCursor={null} nextCursor="abc" pathname="/admin" />);

    expect(screen.queryByRole("link", { name: "Newer posts" })).toBeNull();
    expect(
      screen.getByRole("link", { name: "Older posts" }).getAttribute("href"),
    ).toBe("/admin?cursor=abc");
  });

  it("links only back on the last page", () => {
    render(<Pagination prevCursor="abc" nextCursor={null} />);

    expect(
      screen.getByRole("link", { name: "Newer posts" }).getAttribute("href"),
    ).toBe("/?cursor=abc");
    expect(screen.queryByRole("link", { name: "Older posts" })).toBeNull();
  });

  it("URL-encodes the cursor", () => {
    const cursor = "a+b/c==";
    render(<Pagination prevCursor={cursor} nextCursor={null} />);

    const href = screen
      .getByRole("link", { name: "Newer posts" })
      .getAttribute("href");
    expect(href).not.toBeNull();
    const url = new URL(href!, "http://blog.test");
    expect(url.pathname).toBe("/");
    expect(url.searchParams.get("cursor")).toBe(cursor);
  });

  it("renders nothing when the list is a single page", () => {
    const { container } = render(
      <Pagination prevCursor={null} nextCursor={null} />,
    );

    expect(screen.queryByRole("link")).toBeNull();
    expect(container.innerHTML).toBe("");
  });
});
