import type { Metadata } from "next";
import { Suspense } from "react";

import { BackLink } from "@/components/admin/back-link";
import { PostEditorLoader } from "@/components/admin/post-editor-loader";
import { UnsavedChanges } from "@/components/admin/unsaved-changes";
import { PostEditorSkeleton } from "@/components/skeleton";
import { requireAdmin } from "@/lib/admin/session";
import { listAdminTags } from "@/lib/api/admin";

export const metadata: Metadata = { title: "New post" };

export default function NewPostPage() {
  return (
    <UnsavedChanges>
      <div className="mx-auto max-w-3xl">
        <BackLink />
        <h1 className="mb-6 text-3xl font-bold tracking-tight">New post</h1>
        <Suspense fallback={<PostEditorSkeleton />}>
          <NewPost />
        </Suspense>
      </div>
    </UnsavedChanges>
  );
}

async function NewPost() {
  const admin = await requireAdmin();
  const tags = await listAdminTags(admin);
  return (
    <PostEditorLoader post={null} tagSuggestions={tags.map((t) => t.name)} />
  );
}
