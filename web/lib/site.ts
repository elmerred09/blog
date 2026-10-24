export const siteName = "Bits from Elmer";

// siteUrl is baked into prerendered metadata, so production must set it at
// build time. A later change of domain means rebuilding.
const devSiteURL = "http://localhost:3000";

export function siteUrl(): string {
  const configured = process.env.SITE_URL?.trim().replace(/\/$/, "") ?? "";
  if (configured !== "") {
    return configured;
  }
  if (process.env.NODE_ENV === "production") {
    throw new Error(
      "SITE_URL is not set; a production build must not emit localhost URLs",
    );
  }
  return devSiteURL;
}

// indexingEnabled is true only for the real production deploy. Every other
// environment, including a production build in CI, stays out of search results.
export function indexingEnabled(): boolean {
  return process.env.SITE_INDEXING === "true";
}
