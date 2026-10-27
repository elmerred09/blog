import { PostCard } from "@/components/post-card";
import { Snippet } from "@/components/snippet";
import type { SearchResult } from "@/lib/api";

type Props = {
  query: string;
  results: SearchResult[];
};

export function SearchResultList({ query, results }: Props) {
  if (results.length === 0) {
    return <p className="text-muted">No posts match “{query}”.</p>;
  }
  return (
    <ul className="space-y-5">
      {results.map((result) => (
        <li key={result.slug}>
          <PostCard
            slug={result.slug}
            title={result.title}
            publishedAt={result.published_at}
            tags={result.tags}
          >
            {result.snippet.length > 0 ? (
              <Snippet fragments={result.snippet} />
            ) : (
              <p className="text-sm leading-relaxed text-muted">
                {result.summary}
              </p>
            )}
          </PostCard>
        </li>
      ))}
    </ul>
  );
}
