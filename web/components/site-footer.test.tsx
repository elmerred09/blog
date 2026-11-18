import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { SiteFooter } from "@/components/site-footer";

describe("SiteFooter", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("is the page's footer, with the copyright and social links", () => {
    vi.stubEnv("BUILD_YEAR", "2026");
    render(<SiteFooter />);

    const footer = screen.getByRole("contentinfo");
    expect(footer.textContent).toContain("© 2026 Bits by Elmer");
    expect(
      screen.getByRole("link", { name: "GitHub" }).getAttribute("href"),
    ).toBe("https://github.com/elmerred09");
    expect(screen.getByRole("link", { name: "LinkedIn" })).toBeTruthy();
  });

  it("leaves the year out when the build didn't set one", () => {
    vi.stubEnv("BUILD_YEAR", "");
    render(<SiteFooter />);

    expect(screen.getByRole("contentinfo").textContent).toContain(
      "© Bits by Elmer",
    );
  });
});
