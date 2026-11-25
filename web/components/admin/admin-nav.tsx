"use client";

import { usePathname } from "next/navigation";

import { signOut } from "@/app/sign-in/actions";

// Sign out in the site header, on admin pages only. It goes by the path, not
// the session: proxy.ts sends requests without a session cookie to sign-in
// before any admin page renders, and requireAdmin guards the rest.
export function AdminNav() {
  const pathname = usePathname();
  if (pathname !== "/admin" && !pathname.startsWith("/admin/")) {
    return null;
  }
  return (
    <nav aria-label="Admin">
      <form action={signOut}>
        <button
          type="submit"
          className="cursor-pointer text-sm font-medium text-muted hover:text-foreground"
        >
          Sign out
        </button>
      </form>
    </nav>
  );
}
