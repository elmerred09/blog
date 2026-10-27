import Link from "next/link";
import type { ReactNode } from "react";

import { TagList } from "@/components/tag-list";
import type { Tag } from "@/lib/api";
import { formatDate } from "@/lib/format";

type Props = {
  slug: string;
  title: string;
  publishedAt: string;
  tags: Tag[];
  // The text under the title: the summary, or a search snippet.
  children: ReactNode;
};

// The post card shared by the home list and the search results. The title link
// stretches over the whole card; the tags sit above it (see TagList).
export function PostCard({ slug, title, publishedAt, tags, children }: Props) {
  return (
    <article className="group relative overflow-hidden rounded-2xl border border-line bg-card transition-colors hover:border-line-strong">
      <div className="border-b border-line px-6 py-4">
        <time dateTime={publishedAt} className="text-xs text-subtle">
          {formatDate(publishedAt)}
        </time>
        <h2 className="mt-1 text-xl leading-snug font-semibold tracking-tight">
          <Link
            href={`/posts/${slug}`}
            className="after:absolute after:inset-0 after:content-['']"
          >
            {title}
          </Link>
        </h2>
      </div>
      <div className="space-y-4 px-6 py-5">
        {children}
        <span
          aria-hidden="true"
          className="inline-flex items-center gap-1 text-sm font-medium text-brand"
        >
          Read more
          <svg
            viewBox="0 0 16 16"
            className="size-3.5 transition-transform group-hover:translate-x-0.5"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.75"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M3 8h10M9 4l4 4-4 4" />
          </svg>
        </span>
      </div>
      <TagList tags={tags} className="border-t border-line px-6 py-4" />
    </article>
  );
}
