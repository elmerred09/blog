import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { HeaderSearch, HeaderSearchFromUrl } from "@/components/header-search";

const url = vi.hoisted(() => ({ pathname: "/", search: "" }));

vi.mock("next/navigation", async (importOriginal) => ({
  ...(await importOriginal<typeof import("next/navigation")>()),
  usePathname: () => url.pathname,
  useSearchParams: () => new URLSearchParams(url.search),
}));

describe("HeaderSearch", () => {
  it("submits q to /search as a search landmark", () => {
    render(<HeaderSearch defaultQuery="" />);

    const input = screen.getByRole("searchbox", { name: "Search posts" });
    expect(input.getAttribute("name")).toBe("q");
    expect(input.getAttribute("maxlength")).toBe("200");
    const landmark = screen.getByRole("search");
    expect(landmark.getAttribute("action")).toBe("/search");
    expect(landmark.contains(input)).toBe(true);
  });

  it("links to /search for screens too narrow for the bar", () => {
    render(<HeaderSearch defaultQuery="" />);

    expect(
      screen.getByRole("link", { name: "Search" }).getAttribute("href"),
    ).toBe("/search");
  });

  it("focuses the bar when / is pressed", () => {
    render(<HeaderSearch defaultQuery="" />);

    fireEvent.keyDown(document.body, { key: "/" });

    expect(document.activeElement).toBe(screen.getByRole("searchbox"));
  });

  it.each([
    ["an input", <input key="field" aria-label="field" />],
    [
      "an editor",
      <div key="field" aria-label="field" contentEditable>
        <p>text</p>
      </div>,
    ],
  ])("leaves / alone while typing in %s", (_, field) => {
    render(
      <>
        {field}
        <HeaderSearch defaultQuery="" />
      </>,
    );
    const target = screen.getByLabelText("field");
    const typed = (target.querySelector("p") ?? target) as HTMLElement;

    const notCanceled = fireEvent.keyDown(typed, { key: "/" });

    expect(notCanceled).toBe(true);
    expect(document.activeElement).not.toBe(screen.getByRole("searchbox"));
  });

  it("leaves shortcuts with modifiers alone", () => {
    render(<HeaderSearch defaultQuery="" />);

    fireEvent.keyDown(document.body, { key: "/", ctrlKey: true });

    expect(document.activeElement).not.toBe(screen.getByRole("searchbox"));
  });
});

describe("HeaderSearchFromUrl", () => {
  beforeEach(() => {
    url.pathname = "/";
    url.search = "";
  });

  it.each([
    ["the trimmed query on /search", "/search", "?q=%20keyset%20", "keyset"],
    ["nothing elsewhere", "/posts/keyset", "?q=keyset", ""],
  ])("shows %s", (_, pathname, search, value) => {
    url.pathname = pathname;
    url.search = search;

    render(<HeaderSearchFromUrl />);

    expect(screen.getByRole<HTMLInputElement>("searchbox").value).toBe(value);
  });
});
