import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { AdminNav } from "@/components/admin/admin-nav";

const pathname = vi.hoisted(() => ({ current: "/" }));

vi.mock("next/navigation", () => ({
  usePathname: () => pathname.current,
}));

vi.mock("@/app/sign-in/actions", () => ({
  signOut: vi.fn(),
}));

describe("AdminNav", () => {
  it.each(["/", "/posts/hello", "/administrator", "/sign-in"])(
    "renders nothing on %s",
    (path) => {
      pathname.current = path;
      const { container } = render(<AdminNav />);
      expect(container.innerHTML).toBe("");
    },
  );

  it.each(["/admin", "/admin/posts/new"])(
    "offers the post list and sign-out on %s",
    (path) => {
      pathname.current = path;
      render(<AdminNav />);
      expect(
        screen.getByRole("link", { name: "All posts" }).getAttribute("href"),
      ).toBe("/admin");
      expect(screen.getByRole("button", { name: "Sign out" })).toBeDefined();
    },
  );

  it("marks the post list as current only on it", () => {
    pathname.current = "/admin";
    const { unmount } = render(<AdminNav />);
    expect(
      screen
        .getByRole("link", { name: "All posts" })
        .getAttribute("aria-current"),
    ).toBe("page");
    unmount();

    pathname.current = "/admin/posts/new";
    render(<AdminNav />);
    expect(
      screen
        .getByRole("link", { name: "All posts" })
        .getAttribute("aria-current"),
    ).toBeNull();
  });
});
