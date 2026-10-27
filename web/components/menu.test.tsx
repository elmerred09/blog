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

  it("closes when a link is chosen", () => {
    render(<Menu />);

    fireEvent.click(screen.getByRole("button", { name: "Menu" }));
    fireEvent.click(screen.getByRole("link", { name: "Search" }));

    expect(screen.queryByRole("link", { name: "Search" })).toBeNull();
    expect(screen.getByRole("button", { name: "Menu" })).toBeTruthy();
  });
});
