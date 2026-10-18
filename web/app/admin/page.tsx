import { Suspense } from "react";

import { requireAdmin } from "@/lib/admin/session";

export default function AdminPage() {
  return (
    <>
      <h1 className="mb-6 text-3xl font-bold tracking-tight">Posts</h1>
      <Suspense fallback={<p className="text-zinc-500">Loading…</p>}>
        <AdminHome />
      </Suspense>
    </>
  );
}

// A placeholder until the post list (web-editor) replaces it.
async function AdminHome() {
  const admin = await requireAdmin();
  return (
    <p className="text-zinc-600 dark:text-zinc-400">
      Signed in as {admin.name} (GitHub id {admin.githubId}).
    </p>
  );
}
