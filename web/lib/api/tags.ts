import "server-only";

import { cacheLife, cacheTag } from "next/cache";

import { apiClient, unwrap } from "./client";
import type { Tag, TagPage } from "./types";

// Tags on published posts, sorted by slug. Cached under "posts": saving a
// post is what changes the set, and that already expires this tag.
export async function listTags(): Promise<Tag[]> {
  "use cache";
  cacheLife("minutes");
  cacheTag("posts");

  const result = await apiClient().GET("/api/v1/tags");
  return unwrap(result).tags;
}

export type ListPostsByTagParams = {
  limit?: number;
  cursor?: string;
};

// One page of a tag's published posts, newest first, with the tag itself.
// Returns null on 404 (no published post has the tag, or the cursor is past
// the end) and on 400 (a cursor or limit from an edited URL). Cached like
// listPosts: minutes, under "posts", which a save expires.
export async function listPostsByTag(
  slug: string,
  params: ListPostsByTagParams = {},
): Promise<TagPage | null> {
  "use cache";
  cacheLife("minutes");
  cacheTag("posts");

  const result = await apiClient().GET("/api/v1/tags/{slug}/posts", {
    params: { path: { slug }, query: params },
  });
  if (result.response.status === 400 || result.response.status === 404) {
    return null;
  }
  return unwrap(result);
}
