import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { PostBody } from "@/components/post-body";
import { PostMeta } from "@/components/post-meta";
import { getPost } from "@/lib/api";
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
    <Suspense fallback={<p className="text-zinc-500">Loading post…</p>}>
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
    <article>
      <header className="mb-10 space-y-3">
        <h1 className="text-3xl font-bold tracking-tight">{post.title}</h1>
        <PostMeta publishedAt={post.published_at} tags={post.tags} />
      </header>
      <PostBody html={post.body_html} />
    </article>
  );
}
