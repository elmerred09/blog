import { formatDate, formatReadingTime } from "@/lib/format";

type Props = {
  publishedAt: string;
  readingMinutes: number;
  className?: string;
};

// The publish date and reading time above a post's title, on cards and on the
// post page. Callers set the text size through className.
export function PostDateline({
  publishedAt,
  readingMinutes,
  className = "",
}: Props) {
  return (
    <p className={`flex items-center gap-2 text-subtle ${className}`}>
      <time dateTime={publishedAt}>{formatDate(publishedAt)}</time>
      <span aria-hidden="true" className="size-1 rounded-full bg-brand" />
      <span>{formatReadingTime(readingMinutes)}</span>
    </p>
  );
}
