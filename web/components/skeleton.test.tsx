import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  PostListSkeleton,
  PostSkeleton,
  TopicListSkeleton,
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
