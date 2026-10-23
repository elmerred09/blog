import type { MetadataRoute } from "next";
import { connection } from "next/server";

import { listSitemapEntries } from "@/lib/api";
import { buildSitemap } from "@/lib/sitemap";
import { siteUrl } from "@/lib/site";

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  // connection() waits for a real request. listSitemapEntries has no
  // arguments, so without this next build would call the API.
  await connection();
  const posts = await listSitemapEntries();
  return buildSitemap(
    siteUrl(),
    posts.map((post) => ({
      slug: post.slug,
      lastModified: post.last_modified,
    })),
  );
}
