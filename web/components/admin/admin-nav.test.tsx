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

  it.each(["/admin", "/admin/posts/new", "/admin/tags"])(
    "offers the admin pages and sign-out on %s",
    (path) => {
      pathname.current = path;
      render(<AdminNav />);
      expect(screen.getByRole("button", { name: "Sign out" })).toBeDefined();
      expect(
        screen.getByRole("link", { name: "Posts" }).getAttribute("href"),
      ).toBe("/admin");
      expect(
        screen.getByRole("link", { name: "Tags" }).getAttribute("href"),
      ).toBe("/admin/tags");
    },
  );

  it.each([
    ["/admin", "Posts"],
    ["/admin/posts/new", "Posts"],
    ["/admin/tags", "Tags"],
  ])("marks the current section on %s", (path, current) => {
    pathname.current = path;
    render(<AdminNav />);
    const marked = screen
      .getAllByRole("link")
      .filter((a) => a.getAttribute("aria-current") === "page")
      .map((a) => a.textContent);
    expect(marked).toEqual([current]);
  });
});
