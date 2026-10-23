import { describe, expect, it } from "vitest";

import { buildRobots } from "@/lib/robots";

describe("buildRobots", () => {
  it("disallows the whole site when indexing is off", () => {
    expect(buildRobots(false, "https://blog.example")).toEqual({
      rules: { userAgent: "*", disallow: "/" },
    });
  });

  it("allows the site and points at the sitemap when indexing is on", () => {
    expect(buildRobots(true, "https://blog.example/")).toEqual({
      rules: {
        userAgent: "*",
        allow: "/",
        disallow: ["/admin", "/api/"],
      },
      sitemap: "https://blog.example/sitemap.xml",
    });
  });
});
