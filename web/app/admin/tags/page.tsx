import type { Metadata } from "next";
import { Suspense } from "react";

import { BackLink } from "@/components/admin/back-link";
import { TagColorList } from "@/components/admin/tag-color-list";
import { AdminTagListSkeleton } from "@/components/skeleton";
import { requireAdmin } from "@/lib/admin/session";
import { listAdminTags } from "@/lib/api/admin";

export const metadata: Metadata = { title: "Tags" };

// Laid out like the editor pages: a Back link to the post list over the
// title, in the same column.
export default function AdminTagsPage() {
  return (
    <div className="mx-auto max-w-3xl">
      <BackLink />
      <h1 className="mb-6 text-3xl font-bold tracking-tight">Tags</h1>
      <Suspense fallback={<AdminTagListSkeleton />}>
        <AdminTags />
      </Suspense>
    </div>
  );
}

// Every tag, including ones only drafts use, so a color can be set before
// the post goes out.
async function AdminTags() {
  const admin = await requireAdmin();
  const tags = await listAdminTags(admin);
  return <TagColorList tags={tags} />;
}
