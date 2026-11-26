import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { AdminListFrame } from "@/components/admin/admin-list-frame";
import { AdminPostList } from "@/components/admin/admin-post-list";
import { Pagination } from "@/components/pagination";
import { AdminPostListSkeleton } from "@/components/skeleton";
import { requireAdmin } from "@/lib/admin/session";
import { listAdminPosts } from "@/lib/api/admin";

export default function AdminPage({ searchParams }: PageProps<"/admin">) {
  return (
    <AdminListFrame>
      <div className="mb-8 flex shrink-0 justify-end">
        <h1 className="sr-only">Posts</h1>
        <Link
          href="/admin/posts/new"
          className="inline-flex items-center gap-1.5 rounded-lg bg-brand px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-brand-strong"
        >
          <svg
            viewBox="0 0 16 16"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            aria-hidden="true"
            className="size-3.5"
          >
            <path d="M8 3v10M3 8h10" />
          </svg>
          New post
        </Link>
      </div>
      <div className="max-lg:flex max-lg:min-h-0 max-lg:flex-1 max-lg:flex-col">
        <Suspense fallback={<AdminPostListSkeleton />}>
          <AdminPosts searchParams={searchParams} />
        </Suspense>
      </div>
    </AdminListFrame>
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
    <div className="max-lg:flex max-lg:min-h-0 max-lg:flex-1 max-lg:flex-col">
      <AdminPostList posts={page.items} />
      <div className="max-lg:shrink-0">
        <Pagination
          prevCursor={page.prev_cursor}
          nextCursor={page.next_cursor}
          pathname="/admin"
        />
      </div>
    </div>
  );
}
