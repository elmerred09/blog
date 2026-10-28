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
    expect(screen.getByText(/one bit at a time/)).toBeTruthy();
    expect(screen.getByText("Bits by Elmer").tagName).toBe("STRONG");
  });
});
