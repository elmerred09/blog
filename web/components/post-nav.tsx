import Link from "next/link";

import type { AdjacentPosts } from "@/lib/api";

type Props = {
  adjacent: AdjacentPosts;
};

// The links under a post: the older post on the left, the newer on the right,
// like reading back and forward through the list.
export function PostNav({ adjacent: { newer, older } }: Props) {
  if (newer === null && older === null) {
    return null;
  }
  return (
    <nav
      aria-label="More posts"
      className="mt-16 grid max-w-[70ch] gap-4 border-t border-line pt-8 sm:grid-cols-2"
    >
      {older !== null && <NavLink post={older} direction="older" />}
      {newer !== null && <NavLink post={newer} direction="newer" />}
    </nav>
  );
}

function NavLink({
  post,
  direction,
}: {
  post: { slug: string; title: string };
  direction: "older" | "newer";
}) {
  const label = direction === "older" ? "Older bit" : "Newer bit";
  return (
    <Link
      href={`/posts/${post.slug}`}
      aria-label={`${label}: ${post.title}`}
      className={[
        "group rounded-lg border border-line bg-card p-4 hover:border-line-strong",
        direction === "newer" ? "sm:col-start-2 sm:text-right" : "",
      ].join(" ")}
    >
      <span className="block text-sm text-muted">
        {direction === "older" && <span aria-hidden="true">← </span>}
        {label}
        {direction === "newer" && <span aria-hidden="true"> →</span>}
      </span>
      <span className="mt-1 block font-semibold group-hover:underline">
        {post.title}
      </span>
    </Link>
  );
}
