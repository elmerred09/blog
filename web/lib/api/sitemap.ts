import "server-only";

import { cacheLife, cacheTag } from "next/cache";

import { apiClient, unwrap } from "./client";
import type { SitemapEntry } from "./types";

// Cached for hours under the posts tag, so savePost's updateTag("posts")
// refreshes the sitemap on every publish or edit.
export async function listSitemapEntries(): Promise<SitemapEntry[]> {
  "use cache";
  cacheLife("hours");
  cacheTag("posts");

  const result = await apiClient().GET("/api/v1/sitemap");
  return unwrap(result).items;
}
