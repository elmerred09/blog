"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { signOut } from "@/app/sign-in/actions";

// The admin's links in the site header, on admin pages only. It goes by the
// path, not the session: proxy.ts sends requests without a session cookie to
// sign-in before any admin page renders, and requireAdmin guards the rest.
export function AdminNav() {
  const pathname = usePathname();
  if (pathname !== "/admin" && !pathname.startsWith("/admin/")) {
    return null;
  }
  const link =
    "text-muted hover:text-foreground aria-[current=page]:text-foreground";
  return (
    <nav
      aria-label="Admin"
      className="flex items-center gap-5 text-sm font-medium whitespace-nowrap"
    >
      <Link
        href="/admin"
        aria-current={pathname === "/admin" ? "page" : undefined}
        className={`${link} hidden sm:inline`}
      >
        All posts
      </Link>
      <form action={signOut}>
        <button type="submit" className={`${link} cursor-pointer`}>
          Sign out
        </button>
      </form>
    </nav>
  );
}
