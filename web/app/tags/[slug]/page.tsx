import type { Metadata } from "next";
import { io } from "next/cache";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { Pagination } from "@/components/pagination";
import { PostList } from "@/components/post-list";
import {
  PostListSkeleton,
  TagHeadingSkeleton,
  tagHeadingClass,
  TopicNavSkeleton,
} from "@/components/skeleton";
import { TopicNav } from "@/components/topic-list";
import { listPostsByTag, listTags, type Tag } from "@/lib/api";

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

// Laid out like the devTalk topic page: "Topics * <name>" on one line, a row
// of every topic, then the posts in two columns.
export default function TagPage({ params, searchParams }: Props) {
  return (
    <Suspense
      fallback={
        <>
          <TagHeadingSkeleton />
          <div className="mb-10">
            <TopicNavSkeleton />
          </div>
          <PostListSkeleton columns={2} />
        </>
      }
    >
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
      <header className="mb-10">
        <h1 className={tagHeadingClass}>
          <span>Topics</span>
          <span aria-hidden="true">*</span>
          <span className="min-w-0 break-words">{page.tag.name}</span>
        </h1>
      </header>
      <div className="mb-10">
        <Suspense fallback={<TopicNavSkeleton />}>
          <Topics current={slug} />
        </Suspense>
      </div>
      <PostList posts={page.items} columns={2} />
      <Pagination
        prevCursor={page.prev_cursor}
        nextCursor={page.next_cursor}
        pathname={`/tags/${slug}`}
      />
    </>
  );
}

// The topic row is optional, as on the home page: an API error hides it
// rather than replacing the page with error.tsx.
async function Topics({ current }: { current: string }) {
  // listTags takes no request-derived arguments; io() keeps next build from
  // calling the API to prerender it.
  await io();
  let tags: Tag[];
  try {
    tags = await listTags();
  } catch (err) {
    console.error("list tags for a tag page", err);
    return null;
  }
  return <TopicNav tags={tags} current={current} />;
}
