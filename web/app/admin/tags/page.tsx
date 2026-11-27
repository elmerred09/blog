import type { Metadata } from "next";
import { Suspense } from "react";

import { TagColorList } from "@/components/admin/tag-color-list";
import { AdminTagListSkeleton } from "@/components/skeleton";
import { requireAdmin } from "@/lib/admin/session";
import { listAdminTags } from "@/lib/api/admin";

export const metadata: Metadata = { title: "Tags" };

export default function AdminTagsPage() {
  return (
    <>
      <header className="mb-8 space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Tags</h1>
        <p className="text-sm text-muted">
          Each tag&apos;s chip color on post cards and post pages.
        </p>
      </header>
      <Suspense fallback={<AdminTagListSkeleton />}>
        <AdminTags />
      </Suspense>
    </>
  );
}

// Every tag, including ones only drafts use, so a color can be set before
// the post goes out.
async function AdminTags() {
  const admin = await requireAdmin();
  const tags = await listAdminTags(admin);
  return <TagColorList tags={tags} />;
}
