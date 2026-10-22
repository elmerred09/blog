import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { AdminPostList } from "@/components/admin/admin-post-list";
import { Pagination } from "@/components/pagination";
import { requireAdmin } from "@/lib/admin/session";
import { listAdminPosts } from "@/lib/api/admin";

export default function AdminPage({ searchParams }: PageProps<"/admin">) {
  return (
    <>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-3xl font-bold tracking-tight">Posts</h1>
        <Link
          href="/admin/posts/new"
          className="rounded bg-zinc-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-zinc-700 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-300"
        >
          New post
        </Link>
      </div>
      <Suspense fallback={<p className="text-zinc-500">Loading posts…</p>}>
        <AdminPosts searchParams={searchParams} />
      </Suspense>
    </>
  );
}

// Every post, drafts and scheduled included, newest created first.
async function AdminPosts({
  searchParams,
}: {
  searchParams: PageProps<"/admin">["searchParams"];
}) {
  const admin = await requireAdmin();
  const { cursor } = await searchParams;
  if (Array.isArray(cursor)) {
    notFound();
  }
  const page = await listAdminPosts(admin, { cursor });
  if (page === null) {
    notFound();
  }

  return (
    <>
      <AdminPostList posts={page.items} />
      <Pagination
        prevCursor={page.prev_cursor}
        nextCursor={page.next_cursor}
        pathname="/admin"
      />
    </>
  );
}
