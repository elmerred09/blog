import "server-only";

import { cacheLife, cacheTag } from "next/cache";

import { apiClient, unwrap } from "./client";
import type { SearchResults } from "./types";

export type SearchPostsParams = {
  q: string;
  limit?: number;
};

// Returns null on 400 (a blank or too-long q), like listPosts: a thrown error
// loses its status crossing "use cache".
export async function searchPosts(
  params: SearchPostsParams,
): Promise<SearchResults | null> {
  "use cache";
  cacheLife("minutes");
  cacheTag("posts");

  const result = await apiClient().GET("/api/v1/search", {
    params: { query: params },
  });
  if (result.response.status === 400) {
    return null;
  }
  return unwrap(result);
}
