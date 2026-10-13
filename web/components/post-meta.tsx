import type { Tag } from "@/lib/api";
import { formatDate } from "@/lib/format";

type Props = {
  publishedAt: string;
  tags: Tag[];
};

export function PostMeta({ publishedAt, tags }: Props) {
  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-zinc-500 dark:text-zinc-400">
      <time dateTime={publishedAt}>{formatDate(publishedAt)}</time>
      {tags.length > 0 && (
        <ul aria-label="Tags" className="flex flex-wrap gap-2">
          {tags.map((tag) => (
            <li
              key={tag.slug}
              className="rounded bg-zinc-100 px-1.5 py-0.5 text-xs dark:bg-zinc-800"
            >
              {tag.name}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
