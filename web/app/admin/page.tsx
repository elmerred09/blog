import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense, type ReactNode } from "react";

import { AdminPostList } from "@/components/admin/admin-post-list";
import { Pagination } from "@/components/pagination";
import { requireAdmin } from "@/lib/admin/session";
import { listAdminPosts } from "@/lib/api/admin";

export default function AdminPage({ searchParams }: PageProps<"/admin">) {
  return (
    <Suspense
      fallback={
        <PostsCard>
          <p className="px-6 py-8 text-muted">Loading posts…</p>
        </PostsCard>
      }
    >
      <AdminPosts searchParams={searchParams} />
    </Suspense>
  );
}

// New post sits in the card's header, on its border and the same hairline as
// the column row. The heading stays for the page outline and is not shown.
function PostsCard({ children }: { children: ReactNode }) {
  return (
    <div className="rounded-2xl border border-line">
      <div className="flex items-center justify-end gap-4 border-b border-line px-6 py-4">
        <h1 className="sr-only">Posts</h1>
        <Link
          href="/admin/posts/new"
          className="inline-flex shrink-0 items-center gap-1.5 rounded-lg bg-brand px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-brand-strong"
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
      {children}
    </div>
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
      <PostsCard>
        <AdminPostList posts={page.items} />
      </PostsCard>
      <Pagination
        prevCursor={page.prev_cursor}
        nextCursor={page.next_cursor}
        pathname="/admin"
      />
    </>
  );
}
