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

  it.each(["/admin", "/admin/posts/new"])("offers sign-out on %s", (path) => {
    pathname.current = path;
    render(<AdminNav />);
    expect(screen.getByRole("button", { name: "Sign out" })).toBeDefined();
    expect(screen.queryByRole("link")).toBeNull();
  });
});
