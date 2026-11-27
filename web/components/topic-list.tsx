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
// page: large chips, with the current tag's chip marked and filled black.
export function TopicNav({ tags, current }: Props & { current: string }) {
  if (tags.length === 0) {
    return null;
  }
  return (
    <nav aria-label="Topics">
      <TopicChips tags={tags} current={current} size="large" />
    </nav>
  );
}

// The devTalk template's tag-topic chip in its two sizes. Hover darkens the
// border; the current chip is solid foreground.
const chipSizes = {
  default: "h-10 rounded-lg px-4 text-sm",
  large: "h-20 rounded-2xl px-8 text-base",
};

function TopicChips({
  tags,
  current,
  size = "default",
}: Props & { current?: string; size?: keyof typeof chipSizes }) {
  return (
    <ul className="flex flex-wrap gap-1">
      {tags.map((tag) => {
        const isCurrent = tag.slug === current;
        return (
          <li key={tag.slug}>
            <Link
              href={`/tags/${tag.slug}`}
              aria-current={isCurrent ? "page" : undefined}
              className={`flex items-center border transition-colors duration-200 hover:underline ${chipSizes[size]} ${
                isCurrent
                  ? "border-foreground bg-foreground text-background"
                  : "border-line bg-card text-muted hover:border-subtle"
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
