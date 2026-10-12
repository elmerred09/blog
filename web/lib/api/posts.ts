import "server-only";

import { cacheLife, cacheTag } from "next/cache";

import { apiClient, unwrap } from "./client";
import type { Page, Post } from "./types";

export type ListPostsParams = {
  limit?: number;
  cursor?: string;
};

export async function listPosts(params: ListPostsParams = {}): Promise<Page> {
  "use cache";
  cacheLife("minutes");
  cacheTag("posts");

  return unwrap(
    await apiClient().GET("/api/v1/posts", { params: { query: params } }),
  );
}

// A 404 returns null, and that null is cached like a post: a newly published
// post can 404 until the entry expires or the posts tag is revalidated.
export async function getPost(slug: string): Promise<Post | null> {
  "use cache";
  cacheLife("hours");
  cacheTag("posts", `post:${slug}`);

  const result = await apiClient().GET("/api/v1/posts/{slug}", {
    params: { path: { slug } },
  });
  if (result.response.status === 404) {
    return null;
  }
  return unwrap(result);
}
