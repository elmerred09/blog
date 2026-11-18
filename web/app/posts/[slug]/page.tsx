import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { PostBody } from "@/components/post-body";
import { PostDateline } from "@/components/post-dateline";
import { PostNav } from "@/components/post-nav";
import { PostSkeleton } from "@/components/skeleton";
import { TagList } from "@/components/tag-list";
import { getAdjacentPosts, getPost, type AdjacentPosts } from "@/lib/api";
import { siteName } from "@/lib/site";

type Props = PageProps<"/posts/[slug]">;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { slug } = await params;
  const post = await getPost(slug);
  if (post === null) {
    notFound();
  }
  return {
    title: post.title,
    description: post.summary,
    alternates: { canonical: `/posts/${slug}` },
    // A child's openGraph replaces the layout's, so siteName is repeated.
    openGraph: {
      type: "article",
      siteName,
      title: post.title,
      description: post.summary,
      publishedTime: post.published_at,
      modifiedTime: post.updated_at,
      tags: post.tags.map((tag) => tag.name),
      url: `/posts/${slug}`,
    },
  };
}

export default function PostPage({ params }: Props) {
  return (
    <Suspense fallback={<PostSkeleton />}>
      <PostArticle params={params} />
    </Suspense>
  );
}

async function PostArticle({ params }: { params: Props["params"] }) {
  const { slug } = await params;
  const post = await getPost(slug);
  if (post === null) {
    notFound();
  }

  return (
    <>
      <article>
        <header className="mb-12 space-y-4">
          <PostDateline
            publishedAt={post.published_at}
            readingMinutes={post.reading_minutes}
            className="text-sm"
          />
          <h1 className="text-5xl leading-[1.1] font-semibold tracking-tight">
            {post.title}
          </h1>
          <TagList tags={post.tags} className="pt-2" />
        </header>
        <PostBody html={post.body_html} />
      </article>
      <Suspense>
        <Neighbors slug={slug} />
      </Suspense>
    </>
  );
}

// Optional: an API error hides the links rather than replacing the post with
// error.tsx. Caught here, not in getAdjacentPosts, so the failure isn't cached.
async function Neighbors({ slug }: { slug: string }) {
  let adjacent: AdjacentPosts | null;
  try {
    adjacent = await getAdjacentPosts(slug);
  } catch (err) {
    console.error(`get posts adjacent to ${slug}`, err);
    return null;
  }
  return adjacent === null ? null : <PostNav adjacent={adjacent} />;
}
