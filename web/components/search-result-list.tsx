import Link from "next/link";

import { PostMeta } from "@/components/post-meta";
import { Snippet } from "@/components/snippet";
import type { SearchResult } from "@/lib/api";

type Props = {
  query: string;
  results: SearchResult[];
};

export function SearchResultList({ query, results }: Props) {
  if (results.length === 0) {
    return (
      <p className="text-zinc-500 dark:text-zinc-400">
        No posts match “{query}”.
      </p>
    );
  }
  return (
    <ul className="space-y-10">
      {results.map((result) => (
        <li key={result.slug}>
          <article className="space-y-2">
            <h2 className="text-xl font-semibold tracking-tight">
              <Link
                href={`/posts/${result.slug}`}
                className="hover:underline hover:underline-offset-4"
              >
                {result.title}
              </Link>
            </h2>
            <PostMeta publishedAt={result.published_at} tags={result.tags} />
            {result.snippet.length > 0 ? (
              <Snippet fragments={result.snippet} />
            ) : (
              <p className="text-zinc-700 dark:text-zinc-300">
                {result.summary}
              </p>
            )}
          </article>
        </li>
      ))}
    </ul>
  );
}
