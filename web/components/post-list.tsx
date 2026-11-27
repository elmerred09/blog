import { PostCard } from "@/components/post-card";
import type { PostListItem } from "@/lib/api";

type Props = {
  posts: PostListItem[];
  // 2 lays the cards out in two columns from md up, as on the tag pages.
  columns?: 1 | 2;
};

export function PostList({ posts, columns = 1 }: Props) {
  if (posts.length === 0) {
    return <p className="text-muted">No posts yet.</p>;
  }
  return (
    <ul
      className={
        columns === 2 ? "grid gap-5 md:grid-cols-2" : "flex flex-col gap-5"
      }
    >
      {posts.map((post) => (
        <li key={post.slug} className="flex">
          <PostCard
            slug={post.slug}
            title={post.title}
            publishedAt={post.published_at}
            readingMinutes={post.reading_minutes}
            tags={post.tags}
          >
            <p className="text-sm leading-relaxed text-muted">{post.summary}</p>
          </PostCard>
        </li>
      ))}
    </ul>
  );
}
