import Link from "next/link";

import type { AdminPostListItem } from "@/lib/api/types";
import { formatDateTime } from "@/lib/format";

type State = AdminPostListItem["state"];

const stateLabels: Record<State, string> = {
  draft: "Draft",
  scheduled: "Scheduled",
  published: "Published",
};

const stateStyles: Record<State, string> = {
  draft: "bg-zinc-100 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300",
  scheduled:
    "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300",
  published:
    "bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300",
};

export function StateBadge({ state }: { state: State }) {
  return (
    <span
      data-testid="state"
      className={`rounded px-1.5 py-0.5 text-xs font-medium ${stateStyles[state]}`}
    >
      {stateLabels[state]}
    </span>
  );
}

type Props = {
  posts: AdminPostListItem[];
};

export function AdminPostList({ posts }: Props) {
  if (posts.length === 0) {
    return (
      <p className="text-zinc-600 dark:text-zinc-400">
        No posts yet. Write the first one.
      </p>
    );
  }
  return (
    <table className="w-full text-left text-sm">
      <thead className="border-b border-zinc-200 text-zinc-500 dark:border-zinc-800 dark:text-zinc-400">
        <tr>
          <th className="py-2 pr-4 font-medium">Title</th>
          <th className="py-2 pr-4 font-medium">State</th>
          <th className="py-2 pr-4 font-medium">Published</th>
          <th className="py-2 font-medium">Updated</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-zinc-100 dark:divide-zinc-900">
        {posts.map((post) => (
          <tr key={post.id}>
            <td className="py-3 pr-4">
              <Link
                href={`/admin/posts/${post.id}`}
                className="font-medium text-zinc-900 underline-offset-4 hover:underline dark:text-zinc-100"
              >
                {post.title}
              </Link>
            </td>
            <td className="py-3 pr-4">
              <StateBadge state={post.state} />
            </td>
            <td className="py-3 pr-4 text-zinc-600 dark:text-zinc-400">
              {post.published_at === null
                ? "—"
                : formatDateTime(post.published_at)}
            </td>
            <td className="py-3 text-zinc-600 dark:text-zinc-400">
              {formatDateTime(post.updated_at)}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
