import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Menu } from "@/components/menu";

describe("Menu", () => {
  it("shows Home and Search when opened", () => {
    render(<Menu />);

    expect(screen.queryByRole("link", { name: "Search" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Menu" }));

    expect(
      screen.getByRole("link", { name: "Home" }).getAttribute("href"),
    ).toBe("/");
    expect(
      screen.getByRole("link", { name: "Search" }).getAttribute("href"),
    ).toBe("/search");
    const button = screen.getByRole("button", { name: "Close" });
    expect(button.getAttribute("aria-expanded")).toBe("true");
  });

  it("links to the social profiles when opened", () => {
    render(<Menu />);

    fireEvent.click(screen.getByRole("button", { name: "Menu" }));

    expect(
      screen.getByRole("link", { name: "LinkedIn" }).getAttribute("href"),
    ).toBe("https://www.linkedin.com/in/egonzalez49/");
    expect(
      screen.getByRole("link", { name: "GitHub" }).getAttribute("href"),
    ).toBe("https://github.com/elmerred09");
  });

  it("closes when a link is chosen", () => {
    render(<Menu />);

    fireEvent.click(screen.getByRole("button", { name: "Menu" }));
    fireEvent.click(screen.getByRole("link", { name: "Search" }));

    expect(screen.queryByRole("link", { name: "Search" })).toBeNull();
    expect(screen.getByRole("button", { name: "Menu" })).toBeTruthy();
  });
});
