import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { BackLink } from "@/components/admin/back-link";
import { PostEditorLoader } from "@/components/admin/post-editor-loader";
import { UnsavedChanges } from "@/components/admin/unsaved-changes";
import { PostEditorSkeleton } from "@/components/skeleton";
import { requireAdmin } from "@/lib/admin/session";
import { getAdminPost, listAdminTags } from "@/lib/api/admin";

export const metadata: Metadata = { title: "Edit post" };

export default function EditPostPage({
  params,
}: PageProps<"/admin/posts/[id]">) {
  return (
    <UnsavedChanges>
      <div className="mx-auto max-w-3xl">
        <BackLink />
        <h1 className="mb-6 text-3xl font-bold tracking-tight">Edit post</h1>
        <Suspense fallback={<PostEditorSkeleton />}>
          <EditPost params={params} />
        </Suspense>
      </div>
    </UnsavedChanges>
  );
}

async function EditPost({
  params,
}: {
  params: PageProps<"/admin/posts/[id]">["params"];
}) {
  const admin = await requireAdmin();
  const { id } = await params;
  const [post, tags] = await Promise.all([
    getAdminPost(admin, id),
    listAdminTags(admin),
  ]);
  if (post === null) {
    notFound();
  }
  // Keyed by id, so moving between posts starts a fresh editor rather than
  // keeping the previous post's state and lock token.
  return (
    <PostEditorLoader
      key={post.id}
      post={post}
      tagSuggestions={tags.map((t) => t.name)}
    />
  );
}
