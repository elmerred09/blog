import { notFound } from "next/navigation";
import { Suspense } from "react";

import { Pagination } from "@/components/pagination";
import { PostList } from "@/components/post-list";
import { TopicList } from "@/components/topic-list";
import { listPosts, listTags } from "@/lib/api";

export default function Home({ searchParams }: PageProps<"/">) {
  return (
    <div className="lg:grid lg:grid-cols-[20rem_minmax(0,1fr)] lg:gap-x-10">
      <h1 className="mb-14 text-5xl font-semibold tracking-tight lg:col-start-2">
        Recent Bits
      </h1>
      <div className="mb-14 lg:col-start-1 lg:row-start-2 lg:mb-0">
        <Suspense>
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
  return <TopicList tags={await listTags()} />;
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
