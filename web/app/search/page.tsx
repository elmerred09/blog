import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { SearchForm } from "@/components/search-form";
import { SearchResultList } from "@/components/search-result-list";
import { searchPosts } from "@/lib/api";

// Every ?q= is a distinct URL; keep crawlers from indexing endless result
// pages while still letting them follow the links to posts.
export const metadata: Metadata = {
  title: "Search",
  robots: { index: false },
};

export default function SearchPage({ searchParams }: PageProps<"/search">) {
  return (
    <div className="mx-auto max-w-3xl">
      <h1 className="mb-10 text-5xl font-semibold tracking-tight">Search</h1>
      <Suspense fallback={<SearchForm defaultQuery="" />}>
        <Search searchParams={searchParams} />
      </Suspense>
    </div>
  );
}

async function Search({
  searchParams,
}: {
  searchParams: PageProps<"/search">["searchParams"];
}) {
  const { q } = await searchParams;
  if (Array.isArray(q)) {
    notFound();
  }
  const query = q?.trim() ?? "";

  return (
    <>
      {/* The key resets the input when back/forward changes the query. */}
      <SearchForm key={query} defaultQuery={query} />
      {query !== "" && (
        <div className="mt-10">
          <Suspense fallback={<p className="text-muted">Searching…</p>}>
            <Results query={query} />
          </Suspense>
        </div>
      )}
    </>
  );
}

async function Results({ query }: { query: string }) {
  const results = await searchPosts({ q: query });
  if (results === null) {
    return (
      <p className="text-muted">
        That search can&apos;t be run. Try a shorter one.
      </p>
    );
  }
  return <SearchResultList query={query} results={results.items} />;
}
