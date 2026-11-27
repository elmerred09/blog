"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { signOut } from "@/app/sign-in/actions";

const sections = [
  { href: "/admin", label: "Posts", matches: ["/admin", "/admin/posts"] },
  { href: "/admin/tags", label: "Tags", matches: ["/admin/tags"] },
];

function inSection(pathname: string, matches: string[]): boolean {
  return matches.some((m) =>
    m === "/admin"
      ? pathname === m
      : pathname === m || pathname.startsWith(`${m}/`),
  );
}

// The admin sections and sign-out in the site header, on admin pages only.
// It goes by the path, not the session: proxy.ts sends requests without a
// session cookie to sign-in before any admin page renders, and requireAdmin
// guards the rest.
export function AdminNav() {
  const pathname = usePathname();
  if (pathname !== "/admin" && !pathname.startsWith("/admin/")) {
    return null;
  }
  return (
    <nav aria-label="Admin" className="flex items-center gap-5">
      {sections.map(({ href, label, matches }) => {
        const current = inSection(pathname, matches);
        return (
          <Link
            key={href}
            href={href}
            aria-current={current ? "page" : undefined}
            className={`text-sm font-medium hover:text-foreground ${
              current ? "text-foreground" : "text-muted"
            }`}
          >
            {label}
          </Link>
        );
      })}
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
