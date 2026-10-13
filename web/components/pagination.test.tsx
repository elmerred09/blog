import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Pagination } from "@/components/pagination";

describe("Pagination", () => {
  it("links to the next page with the cursor", () => {
    render(<Pagination nextCursor="MjAyNi0xMC0wMQ" />);

    const link = screen.getByRole("link", { name: "Older posts" });
    expect(link.getAttribute("href")).toBe("/?cursor=MjAyNi0xMC0wMQ");
  });

  it("URL-encodes the cursor", () => {
    const cursor = "a+b/c==";
    render(<Pagination nextCursor={cursor} />);

    const href = screen
      .getByRole("link", { name: "Older posts" })
      .getAttribute("href");
    expect(href).not.toBeNull();
    const url = new URL(href!, "http://blog.test");
    expect(url.pathname).toBe("/");
    expect(url.searchParams.get("cursor")).toBe(cursor);
  });

  it("renders nothing on the last page", () => {
    const { container } = render(<Pagination nextCursor={null} />);

    expect(screen.queryByRole("link")).toBeNull();
    expect(container.innerHTML).toBe("");
  });
});
