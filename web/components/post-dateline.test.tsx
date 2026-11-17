import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PostDateline } from "@/components/post-dateline";

describe("PostDateline", () => {
  it("shows the date as a time element and the reading time", () => {
    render(
      <PostDateline publishedAt="2026-10-02T12:00:00Z" readingMinutes={7} />,
    );

    const time = screen.getByText("Oct 2, 2026");
    expect(time.tagName).toBe("TIME");
    expect(time.getAttribute("datetime")).toBe("2026-10-02T12:00:00Z");
    expect(screen.getByText("7 min read")).toBeTruthy();
  });

  it("separates them with a dot that has no text and is hidden from screen readers", () => {
    const { container } = render(
      <PostDateline publishedAt="2026-10-02T12:00:00Z" readingMinutes={1} />,
    );

    const separator = container.querySelector('[aria-hidden="true"]');
    expect(separator).not.toBeNull();
    expect(separator?.textContent).toBe("");
    expect(container.textContent).toBe("Oct 2, 20261 min read");
  });
});
