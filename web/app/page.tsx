import { io } from "next/cache";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { Pagination } from "@/components/pagination";
import { PostList } from "@/components/post-list";
import { TopicList } from "@/components/topic-list";
import { listPosts, listTags, type Tag } from "@/lib/api";

export default function Home({ searchParams }: PageProps<"/">) {
  return (
    <div className="lg:grid lg:grid-cols-[20rem_minmax(0,1fr)] lg:gap-x-10">
      <h1 className="mb-14 text-5xl font-semibold tracking-tight lg:col-start-2">
        Recent Bits
      </h1>
      <div className="mb-14 lg:col-start-1 lg:row-start-2 lg:mb-0">
        <Suspense fallback={<p className="text-muted">Loading topics…</p>}>
          <Topics />
        </Suspense>
      </div>
      <div className="lg:col-start-2 lg:row-start-2">
        <Suspense fallback={<p className="text-muted">Loading posts…</p>}>
          <Posts searchParams={searchParams} />
        </Suspense>
      </div>
    </div>
  );
}

async function Topics() {
  // listTags has no arguments, so next build would call the API to prerender
  // it. io() stops the prerender here but still lets prefetches include it.
  await io();
  // The column is optional: an API error hides it rather than replacing the
  // page with error.tsx. Caught here, not in listTags, so the failure isn't
  // cached.
  let tags: Tag[];
  try {
    tags = await listTags();
  } catch (err) {
    console.error("list tags for the home page", err);
    return null;
  }
  return <TopicList tags={tags} />;
}

async function Posts({
  searchParams,
}: {
  searchParams: PageProps<"/">["searchParams"];
}) {
  const { cursor } = await searchParams;
  if (Array.isArray(cursor)) {
    notFound();
  }
  const page = await listPosts({ cursor });
  if (page === null) {
    notFound();
  }

  return (
    <>
      <PostList posts={page.items} />
      <Pagination prevCursor={page.prev_cursor} nextCursor={page.next_cursor} />
    </>
  );
}
