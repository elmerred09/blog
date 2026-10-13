import Link from "next/link";

type Props = {
  nextCursor: string | null;
};

export function Pagination({ nextCursor }: Props) {
  if (nextCursor === null) {
    return null;
  }
  return (
    <nav aria-label="Pagination" className="mt-12">
      <Link
        href={{ pathname: "/", query: { cursor: nextCursor } }}
        className="font-medium text-zinc-900 underline-offset-4 hover:underline dark:text-zinc-100"
      >
        Older posts
      </Link>
    </nav>
  );
}
