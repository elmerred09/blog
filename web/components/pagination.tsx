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
          className="font-medium text-zinc-900 underline-offset-4 hover:underline dark:text-zinc-100"
        >
          Newer posts
        </Link>
      )}
      {nextCursor !== null && (
        <Link
          href={{ pathname, query: { cursor: nextCursor } }}
          className="font-medium text-zinc-900 underline-offset-4 hover:underline dark:text-zinc-100"
        >
          Older posts
        </Link>
      )}
    </nav>
  );
}
