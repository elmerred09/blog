import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";

import { requireAdmin } from "@/lib/admin/session";

import { signOut } from "../sign-in/actions";

export const metadata: Metadata = {
  title: { default: "Admin", template: "%s | Admin" },
  robots: { index: false, follow: false },
};

// The session read sits in its own boundary, so the layout itself stays in
// the static shell. Pages still call requireAdmin themselves: a layout check
// doesn't guard a page's data or its server actions.
export default function AdminLayout({ children }: LayoutProps<"/admin">) {
  return (
    <>
      <div className="mb-8 flex items-center justify-between border-b border-zinc-200 pb-4 text-sm dark:border-zinc-800">
        <Link href="/admin" className="font-medium">
          Editor
        </Link>
        <Suspense fallback={null}>
          <AdminBar />
        </Suspense>
      </div>
      {children}
    </>
  );
}

async function AdminBar() {
  const admin = await requireAdmin();
  return (
    <div className="flex items-center gap-3">
      <span className="text-zinc-600 dark:text-zinc-400">{admin.name}</span>
      <form action={signOut}>
        <button
          type="submit"
          className="font-medium text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
        >
          Sign out
        </button>
      </form>
    </div>
  );
}
