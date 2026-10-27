import Link from "next/link";

import type { Tag } from "@/lib/api";

type Props = {
  tags: Tag[];
};

// The home page's topic column, styled after the devTalk template: a heading
// and a wrapping cloud of neutral chips. Each chip links home until the tag
// filter exists.
export function TopicList({ tags }: Props) {
  if (tags.length === 0) {
    return null;
  }
  return (
    <section aria-labelledby="topics-heading">
      <h2
        id="topics-heading"
        className="mb-6 text-2xl font-semibold tracking-tight"
      >
        Dive in by topic
      </h2>
      <ul className="flex flex-wrap gap-1">
        {tags.map((tag) => (
          <li key={tag.slug}>
            <Link
              href="/"
              className="flex h-10 items-center rounded-lg border border-line bg-card px-4 text-sm text-muted hover:underline"
            >
              {tag.name}
            </Link>
          </li>
        ))}
      </ul>
    </section>
  );
}
