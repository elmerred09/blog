import Link from "next/link";

import type { Tag } from "@/lib/api";

type Props = {
  tags: Tag[];
};

// The home page's topic column, styled after the devTalk template: a heading
// and a wrapping cloud of neutral chips. Each chip links to its tag's page.
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
      <TopicChips tags={tags} />
    </section>
  );
}

// The row of every topic above a tag page's posts, as on the devTalk topic
// page. The current tag's chip is marked and filled.
export function TopicNav({ tags, current }: Props & { current: string }) {
  if (tags.length === 0) {
    return null;
  }
  return (
    <nav aria-label="Topics">
      <TopicChips tags={tags} current={current} />
    </nav>
  );
}

function TopicChips({ tags, current }: Props & { current?: string }) {
  return (
    <ul className="flex flex-wrap gap-1">
      {tags.map((tag) => {
        const isCurrent = tag.slug === current;
        return (
          <li key={tag.slug}>
            <Link
              href={`/tags/${tag.slug}`}
              aria-current={isCurrent ? "page" : undefined}
              className={`flex h-10 items-center rounded-lg border px-4 text-sm hover:underline ${
                isCurrent
                  ? "border-brand bg-chip font-medium text-brand-strong"
                  : "border-line bg-card text-muted"
              }`}
            >
              {tag.name}
            </Link>
          </li>
        );
      })}
    </ul>
  );
}
