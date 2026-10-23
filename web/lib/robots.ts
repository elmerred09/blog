import type { MetadataRoute } from "next";

// buildRobots allows the public site only when indexing is on. Otherwise
// the whole site is disallowed, which is every environment but production.
export function buildRobots(
  indexing: boolean,
  siteURL: string,
): MetadataRoute.Robots {
  if (!indexing) {
    return { rules: { userAgent: "*", disallow: "/" } };
  }
  const base = siteURL.replace(/\/$/, "");
  return {
    rules: {
      userAgent: "*",
      allow: "/",
      disallow: ["/admin", "/api/"],
    },
    sitemap: `${base}/sitemap.xml`,
  };
}
