import Link from "next/link";

import { PostMeta } from "@/components/post-meta";
import type { PostListItem } from "@/lib/api";

type Props = {
  posts: PostListItem[];
};

export function PostList({ posts }: Props) {
  if (posts.length === 0) {
    return <p className="text-zinc-500 dark:text-zinc-400">No posts yet.</p>;
  }
  return (
    <ul className="space-y-10">
      {posts.map((post) => (
        <li key={post.slug}>
          <article className="space-y-2">
            <h2 className="text-xl font-semibold tracking-tight">
              <Link
                href={`/posts/${post.slug}`}
                className="hover:underline hover:underline-offset-4"
              >
                {post.title}
              </Link>
            </h2>
            <PostMeta publishedAt={post.published_at} tags={post.tags} />
            <p className="text-zinc-700 dark:text-zinc-300">{post.summary}</p>
          </article>
        </li>
      ))}
    </ul>
  );
}
