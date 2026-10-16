import "server-only";

import { cacheLife, cacheTag } from "next/cache";

import { apiClient, unwrap } from "./client";
import type { Page, Post } from "./types";

export type ListPostsParams = {
  limit?: number;
  cursor?: string;
};

// Expected outcomes are return values, not errors: only values cross the
// "use cache" boundary intact. A thrown ApiError reaches the caller as a plain
// Error with a digest, so callers can't branch on its status.

// Returns null on 400: a cursor or limit the API rejects, which can only come
// from an edited URL.
export async function listPosts(
  params: ListPostsParams = {},
): Promise<Page | null> {
  "use cache";
  cacheLife("minutes");
  cacheTag("posts");

  const result = await apiClient().GET("/api/v1/posts", {
    params: { query: params },
  });
  if (result.response.status === 400) {
    return null;
  }
  return unwrap(result);
}

// A 404 returns null, cached for minutes rather than hours: a scheduled post
// 404s until its published_at, and should appear soon after. cacheLife is
// called once per branch, after the fetch.
export async function getPost(slug: string): Promise<Post | null> {
  "use cache";
  cacheTag("posts", `post:${slug}`);

  const result = await apiClient().GET("/api/v1/posts/{slug}", {
    params: { path: { slug } },
  });
  if (result.response.status === 404) {
    cacheLife("minutes");
    return null;
  }
  const post = unwrap(result);
  cacheLife("hours");
  return post;
}
