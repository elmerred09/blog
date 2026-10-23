import { describe, expect, it } from "vitest";

import { buildSitemap } from "@/lib/sitemap";

const site = "https://blog.example";

describe("buildSitemap", () => {
  it("lists the home page and each post at an absolute URL", () => {
    const entries = buildSitemap(site, [
      { slug: "older", lastModified: "2020-01-01T12:00:00Z" },
      { slug: "newer", lastModified: "2026-10-02T08:30:00Z" },
    ]);

    expect(entries).toEqual([
      { url: `${site}/`, lastModified: "2026-10-02T08:30:00Z" },
      {
        url: `${site}/posts/older`,
        lastModified: "2020-01-01T12:00:00Z",
      },
      {
        url: `${site}/posts/newer`,
        lastModified: "2026-10-02T08:30:00Z",
      },
    ]);
  });

  it("omits a last-modified time when nothing is published", () => {
    expect(buildSitemap(`${site}/`, [])).toEqual([{ url: `${site}/` }]);
  });
});
