import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import "@/app/chroma.css";
import { PostMeta } from "@/components/post-meta";
import { getPost } from "@/lib/api";

type Props = PageProps<"/posts/[slug]">;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { slug } = await params;
  const post = await getPost(slug);
  if (post === null) {
    notFound();
  }
  return { title: post.title, description: post.summary };
}

export default function PostPage({ params }: Props) {
  return (
    <Suspense fallback={<p className="text-zinc-500">Loading post…</p>}>
      <PostBody params={params} />
    </Suspense>
  );
}

async function PostBody({ params }: { params: Props["params"] }) {
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
      {/* body_html comes from our markdown renderer, which escapes raw HTML. */}
      <div
        className="prose max-w-none prose-zinc dark:prose-invert prose-pre:text-inherit"
        dangerouslySetInnerHTML={{ __html: post.body_html }}
      />
    </article>
  );
}
