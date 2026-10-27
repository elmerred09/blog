import "server-only";

import { cacheLife, cacheTag } from "next/cache";

import { apiClient, unwrap } from "./client";
import type { Tag } from "./types";

// Tags on published posts, sorted by slug. Cached under "posts": saving a
// post is what changes the set, and that already expires this tag.
export async function listTags(): Promise<Tag[]> {
  "use cache";
  cacheLife("minutes");
  cacheTag("posts");

  const result = await apiClient().GET("/api/v1/tags");
  return unwrap(result).tags;
}
