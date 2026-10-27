import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { PostBody } from "@/components/post-body";
import { TagList } from "@/components/tag-list";
import { getPost } from "@/lib/api";
import { formatDate } from "@/lib/format";
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
    <Suspense fallback={<p className="text-muted">Loading post…</p>}>
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
    <article className="mx-auto max-w-3xl">
      <header className="mb-12 space-y-4">
        <time
          dateTime={post.published_at}
          className="block text-xs text-subtle"
        >
          {formatDate(post.published_at)}
        </time>
        <h1 className="text-5xl leading-[1.1] font-semibold tracking-tight">
          {post.title}
        </h1>
        <TagList tags={post.tags} className="pt-2" />
      </header>
      <PostBody html={post.body_html} />
    </article>
  );
}
