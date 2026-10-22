import { notFound } from "next/navigation";
import { Suspense } from "react";

import { Pagination } from "@/components/pagination";
import { PostList } from "@/components/post-list";
import { listPosts } from "@/lib/api";

export default function Home({ searchParams }: PageProps<"/">) {
  return (
    <>
      <h1 className="mb-10 text-3xl font-bold tracking-tight">Posts</h1>
      <Suspense fallback={<p className="text-zinc-500">Loading posts…</p>}>
        <Posts searchParams={searchParams} />
      </Suspense>
    </>
  );
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
