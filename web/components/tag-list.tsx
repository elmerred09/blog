import Link from "next/link";

import type { Tag } from "@/lib/api";

type Props = {
  tags: Tag[];
  className?: string;
};

// Each tag links to its tag's page. relative z-10 keeps the link clickable
// above a card's stretched title link.
export function TagList({ tags, className = "" }: Props) {
  if (tags.length === 0) {
    return null;
  }
  return (
    <ul aria-label="Tags" className={`flex flex-wrap gap-2 ${className}`}>
      {tags.map((tag) => (
        <li key={tag.slug}>
          <Link
            href={`/tags/${tag.slug}`}
            className="relative z-10 block rounded-md bg-chip px-2.5 py-1 text-xs font-medium text-brand-strong hover:underline"
          >
            {tag.name}
          </Link>
        </li>
      ))}
    </ul>
  );
}
