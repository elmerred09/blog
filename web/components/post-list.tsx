import { PostCard } from "@/components/post-card";
import type { PostListItem } from "@/lib/api";

type Props = {
  posts: PostListItem[];
};

export function PostList({ posts }: Props) {
  if (posts.length === 0) {
    return <p className="text-muted">No posts yet.</p>;
  }
  return (
    <ul className="space-y-5">
      {posts.map((post) => (
        <li key={post.slug}>
          <PostCard
            slug={post.slug}
            title={post.title}
            publishedAt={post.published_at}
            tags={post.tags}
          >
            <p className="text-sm leading-relaxed text-muted">{post.summary}</p>
          </PostCard>
        </li>
      ))}
    </ul>
  );
}
