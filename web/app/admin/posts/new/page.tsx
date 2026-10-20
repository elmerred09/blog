import type { Metadata } from "next";
import { Suspense } from "react";

import { PostEditorLoader } from "@/components/admin/post-editor-loader";
import { requireAdmin } from "@/lib/admin/session";
import { listAdminTags } from "@/lib/api/admin";

export const metadata: Metadata = { title: "New post" };

export default function NewPostPage() {
  return (
    <>
      <h1 className="mb-6 text-3xl font-bold tracking-tight">New post</h1>
      <Suspense fallback={<p className="text-zinc-500">Loading editor…</p>}>
        <NewPost />
      </Suspense>
    </>
  );
}

async function NewPost() {
  const admin = await requireAdmin();
  const tags = await listAdminTags(admin);
  return (
    <PostEditorLoader post={null} tagSuggestions={tags.map((t) => t.name)} />
  );
}
