import type { MetadataRoute } from "next";

export type SitemapPost = {
  slug: string;
  lastModified: string;
};

// buildSitemap is the home page, one URL per post, then one per tag page.
// The home page's last-modified time is the newest post, so a publish shows
// up there too. posts may arrive in any order. Tag pages carry no
// last-modified time: the API doesn't say when a tag's post list changed,
// and a wrong date is worse than none.
export function buildSitemap(
  siteURL: string,
  posts: SitemapPost[],
  tagSlugs: string[] = [],
): MetadataRoute.Sitemap {
  const base = siteURL.replace(/\/$/, "");
  const home: MetadataRoute.Sitemap[number] = { url: `${base}/` };
  const newest = posts.reduce<string | undefined>((latest, post) => {
    if (latest === undefined || post.lastModified > latest) {
      return post.lastModified;
    }
    return latest;
  }, undefined);
  if (newest !== undefined) {
    home.lastModified = newest;
  }
  return [
    home,
    ...posts.map((post) => ({
      url: `${base}/posts/${post.slug}`,
      lastModified: post.lastModified,
    })),
    ...tagSlugs.map((slug) => ({ url: `${base}/tags/${slug}` })),
  ];
}
