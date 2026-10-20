import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { PostEditorLoader } from "@/components/admin/post-editor-loader";
import { requireAdmin } from "@/lib/admin/session";
import { getAdminPost, listAdminTags } from "@/lib/api/admin";

export const metadata: Metadata = { title: "Edit post" };

export default function EditPostPage({
  params,
}: PageProps<"/admin/posts/[id]">) {
  return (
    <>
      <h1 className="mb-6 text-3xl font-bold tracking-tight">Edit post</h1>
      <Suspense fallback={<p className="text-zinc-500">Loading editor…</p>}>
        <EditPost params={params} />
      </Suspense>
    </>
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
