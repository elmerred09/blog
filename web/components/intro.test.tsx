import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Intro } from "@/components/intro";

describe("Intro", () => {
  it("leads with the statement as the page heading", () => {
    render(<Intro />);

    expect(
      screen.getByRole("heading", {
        level: 1,
        name: "Making sense of software, bit by bit.",
      }),
    ).toBeTruthy();
    expect(
      screen.getByText(/software engineer with six years of experience/),
    ).toBeTruthy();
  });

  it("links to the social profiles", () => {
    render(<Intro />);

    expect(
      screen.getByRole("link", { name: "LinkedIn" }).getAttribute("href"),
    ).toBe("https://www.linkedin.com/in/egonzalez49/");
    expect(
      screen.getByRole("link", { name: "GitHub" }).getAttribute("href"),
    ).toBe("https://github.com/elmerred09");
  });
});
