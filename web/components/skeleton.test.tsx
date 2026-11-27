import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  AdminPostListSkeleton,
  PostEditorSkeleton,
  PostListSkeleton,
  PostSkeleton,
  TagHeadingSkeleton,
  TopicListSkeleton,
  TopicNavSkeleton,
} from "@/components/skeleton";

describe("TopicListSkeleton", () => {
  it("keeps the heading and announces that topics are loading", () => {
    render(<TopicListSkeleton />);

    expect(
      screen.getByRole("heading", { name: "Dive in by topic" }),
    ).toBeTruthy();
    expect(screen.getByRole("status").textContent).toBe("Loading topics…");
  });
});

describe("PostSkeleton", () => {
  it("announces that the post is loading", () => {
    render(<PostSkeleton />);

    expect(screen.getByRole("status").textContent).toBe("Loading post…");
  });
});

describe("PostListSkeleton", () => {
  it("announces that posts are loading", () => {
    render(<PostListSkeleton />);

    expect(screen.getByRole("status").textContent).toBe("Loading posts…");
  });
});

describe("AdminPostListSkeleton", () => {
  it("keeps the column names and announces that posts are loading", () => {
    render(<AdminPostListSkeleton />);

    expect(screen.getByRole("status").textContent).toBe("Loading posts…");
    expect(screen.getByRole("columnheader", { name: "Title" })).toBeTruthy();
    expect(screen.getByRole("columnheader", { name: "Status" })).toBeTruthy();
    expect(
      screen.getByRole("columnheader", { name: "Published" }),
    ).toBeTruthy();
    expect(screen.getByRole("columnheader", { name: "Updated" })).toBeTruthy();
  });
});

describe("PostEditorSkeleton", () => {
  it("keeps the field labels and announces that the editor is loading", () => {
    render(<PostEditorSkeleton />);

    expect(screen.getByRole("status").textContent).toBe("Loading editor…");
    // Labels are visual only: the fields are not real inputs yet.
    expect(screen.getByText("Title")).toBeTruthy();
    expect(screen.getByText("Publishing")).toBeTruthy();
    expect(screen.queryByRole("textbox")).toBeNull();
  });
});

describe("TagHeadingSkeleton", () => {
  it("keeps the label and hides the placeholder name", () => {
    const { container } = render(<TagHeadingSkeleton />);

    expect(screen.getByText("Topics")).toBeTruthy();
    expect(screen.queryByRole("heading")).toBeNull();
    expect(container.querySelector('[aria-hidden="true"]')).toBeTruthy();
  });
});

describe("TopicNavSkeleton", () => {
  it("is hidden from assistive tech", () => {
    const { container } = render(<TopicNavSkeleton />);

    expect(container.firstElementChild?.getAttribute("aria-hidden")).toBe(
      "true",
    );
  });
});
