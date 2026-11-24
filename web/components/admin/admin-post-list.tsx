import Link from "next/link";

import { LocalTime } from "@/components/admin/local-time";
import type { AdminPostListItem } from "@/lib/api/types";

type State = AdminPostListItem["state"];

const stateLabels: Record<State, string> = {
  draft: "Draft",
  scheduled: "Scheduled",
  published: "Published",
};

const stateStyles: Record<State, string> = {
  draft: "bg-line text-muted",
  scheduled: "bg-amber-100 text-amber-800",
  published: "bg-chip text-brand-strong",
};

export function StateBadge({ state }: { state: State }) {
  return (
    <span
      data-testid="state"
      className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${stateStyles[state]}`}
    >
      {stateLabels[state]}
    </span>
  );
}

type Props = {
  posts: AdminPostListItem[];
};

const iconButton =
  "flex size-10 items-center justify-center rounded-lg border border-line bg-background text-muted";

export function AdminPostList({ posts }: Props) {
  if (posts.length === 0) {
    return <p className="text-muted">No posts yet. Write the first one.</p>;
  }
  return (
    <div className="overflow-x-auto rounded-2xl border border-line">
      <table className="w-full min-w-3xl table-fixed text-left text-sm">
        <thead className="border-b border-line bg-card text-muted">
          <tr>
            <th className="px-6 py-3 font-medium">Title</th>
            <th className="w-32 px-4 py-3 font-medium">Status</th>
            <th className="w-40 px-4 py-3 font-medium">Published</th>
            <th className="w-40 px-4 py-3 font-medium">Updated</th>
            <th className="w-36 px-6 py-3">
              <span className="sr-only">Actions</span>
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-line">
          {posts.map((post) => (
            <tr
              key={post.id}
              className="transition-colors hover:bg-chip/40 has-focus-visible:bg-chip/40"
            >
              <td className="truncate px-6 py-4">
                <Link
                  href={`/admin/posts/${post.id}`}
                  className="font-medium underline-offset-4 hover:underline"
                >
                  {post.title}
                </Link>
              </td>
              <td className="px-4 py-4">
                <StateBadge state={post.state} />
              </td>
              <td className="px-4 py-4 whitespace-nowrap text-muted">
                {post.published_at === null ? (
                  "—"
                ) : (
                  <LocalTime iso={post.published_at} />
                )}
              </td>
              <td className="px-4 py-4 whitespace-nowrap text-muted">
                <LocalTime iso={post.updated_at} />
              </td>
              <td className="px-6 py-4">
                <div className="flex justify-end gap-2">
                  <Link
                    href={`/admin/posts/${post.id}`}
                    aria-label={`Edit ${post.title}`}
                    title="Edit"
                    className={`${iconButton} hover:border-line-strong hover:text-foreground`}
                  >
                    <PencilIcon />
                  </Link>
                  <button
                    type="button"
                    aria-label={`Delete ${post.title}`}
                    aria-disabled="true"
                    title="Delete isn't available yet"
                    className={`${iconButton} cursor-not-allowed`}
                  >
                    <TrashIcon />
                  </button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function PencilIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      className="size-4"
    >
      <path d="M4 20h4L18.5 9.5a2.12 2.12 0 0 0-3-3L5 17v3Z" />
      <path d="m13.5 6.5 3 3" />
    </svg>
  );
}

function TrashIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      className="size-4"
    >
      <path d="M4 7h16M10 11v6M14 11v6" />
      <path d="M6 7l1 12a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-12" />
      <path d="M9 7V4h6v3" />
    </svg>
  );
}
