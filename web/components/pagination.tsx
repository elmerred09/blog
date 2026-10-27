import Link from "next/link";

type Props = {
  prevCursor: string | null;
  nextCursor: string | null;
  pathname?: string;
};

export function Pagination({ prevCursor, nextCursor, pathname = "/" }: Props) {
  if (prevCursor === null && nextCursor === null) {
    return null;
  }
  return (
    <nav aria-label="Pagination" className="mt-12 flex gap-6">
      {prevCursor !== null && (
        <Link
          href={{ pathname, query: { cursor: prevCursor } }}
          className="rounded-full border border-line bg-card px-4 py-1.5 text-sm font-medium hover:border-line-strong"
        >
          Newer posts
        </Link>
      )}
      {nextCursor !== null && (
        <Link
          href={{ pathname, query: { cursor: nextCursor } }}
          className="rounded-full border border-line bg-card px-4 py-1.5 text-sm font-medium hover:border-line-strong"
        >
          Older posts
        </Link>
      )}
    </nav>
  );
}
