import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { SearchForm } from "@/components/search-form";

describe("SearchForm", () => {
  it("submits q to /search", () => {
    render(<SearchForm defaultQuery="" />);

    const input = screen.getByRole("searchbox", { name: "Search posts" });
    expect(input.getAttribute("name")).toBe("q");
    expect(input.closest("form")?.getAttribute("action")).toBe("/search");
  });

  it("is a search landmark", () => {
    render(<SearchForm defaultQuery="" />);

    const landmark = screen.getByRole("search");
    expect(landmark.tagName).toBe("FORM");
    expect(landmark.contains(screen.getByRole("searchbox"))).toBe(true);
  });

  it("starts with the current query", () => {
    render(<SearchForm defaultQuery="keyset pagination" />);

    const input = screen.getByRole<HTMLInputElement>("searchbox");
    expect(input.value).toBe("keyset pagination");
  });

  it("caps the input at the API's maximum length", () => {
    render(<SearchForm defaultQuery="" />);

    expect(screen.getByRole("searchbox").getAttribute("maxlength")).toBe("200");
  });
});
