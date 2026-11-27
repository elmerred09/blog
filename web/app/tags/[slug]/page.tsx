import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { Pagination } from "@/components/pagination";
import { PostList } from "@/components/post-list";
import { PostListSkeleton, TagHeadingSkeleton } from "@/components/skeleton";
import { listPostsByTag } from "@/lib/api";

type Props = PageProps<"/tags/[slug]">;

// Reads only the first page, which the page itself shares from the cache.
// Every cursor page has the same canonical URL: the tag's first page.
export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { slug } = await params;
  const page = await listPostsByTag(slug);
  if (page === null) {
    notFound();
  }
  return {
    title: page.tag.name,
    description: `Posts about ${page.tag.name}, newest first.`,
    alternates: { canonical: `/tags/${slug}` },
  };
}

export default function TagPage({ params, searchParams }: Props) {
  return (
    <Suspense fallback={<TagPageSkeleton />}>
      <TagPosts params={params} searchParams={searchParams} />
    </Suspense>
  );
}

async function TagPosts({
  params,
  searchParams,
}: {
  params: Props["params"];
  searchParams: Props["searchParams"];
}) {
  const [{ slug }, { cursor }] = await Promise.all([params, searchParams]);
  if (Array.isArray(cursor)) {
    notFound();
  }
  const page = await listPostsByTag(slug, { cursor });
  if (page === null) {
    notFound();
  }

  return (
    <>
      <TagHeading name={page.tag.name} />
      <PostList posts={page.items} />
      <Pagination
        prevCursor={page.prev_cursor}
        nextCursor={page.next_cursor}
        pathname={`/tags/${slug}`}
      />
    </>
  );
}

function TagHeading({ name }: { name: string }) {
  return (
    <header className="mb-10 space-y-2">
      <p className="text-sm font-medium text-muted">Topic</p>
      <h1 className="text-5xl font-semibold tracking-tight break-words">
        {name}
      </h1>
    </header>
  );
}

function TagPageSkeleton() {
  return (
    <>
      <TagHeadingSkeleton />
      <PostListSkeleton />
    </>
  );
}
