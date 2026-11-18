import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { SearchForm, SearchFormFromUrl } from "@/components/search-form";
import { SearchResultList } from "@/components/search-result-list";
import { searchPosts } from "@/lib/api";

// Every ?q= is a distinct URL; keep crawlers from indexing endless result
// pages while still letting them follow the links to posts.
export const metadata: Metadata = {
  title: "Search",
  robots: { index: false },
};

type SearchParams = PageProps<"/search">["searchParams"];

export default function SearchPage({ searchParams }: PageProps<"/search">) {
  return (
    <div className="max-w-3xl">
      <Suspense fallback={<SearchHeading query="" />}>
        <Heading searchParams={searchParams} />
      </Suspense>
      {/* The header's search bar is the form on wider screens; below sm it's
          only an icon, so the page shows its own form there. It sits outside
          the boundaries that wait on searchParams, so streamed content can't
          replace it while someone is typing. */}
      <div className="mb-10 sm:hidden">
        <Suspense fallback={<SearchForm defaultQuery="" />}>
          <SearchFormFromUrl />
        </Suspense>
      </div>
      <Suspense>
        <Search searchParams={searchParams} />
      </Suspense>
    </div>
  );
}

async function readQuery(searchParams: SearchParams) {
  const { q } = await searchParams;
  if (Array.isArray(q)) {
    notFound();
  }
  return q?.trim() ?? "";
}

async function Heading({ searchParams }: { searchParams: SearchParams }) {
  return <SearchHeading query={await readQuery(searchParams)} />;
}

function SearchHeading({ query }: { query: string }) {
  return (
    <>
      <h1 className="mb-10 text-5xl font-semibold tracking-tight break-words">
        {query === "" ? "Search" : `Results for “${query}”`}
      </h1>
      {query === "" && (
        <p className="hidden text-muted sm:block">
          Type in the search bar above, or press <kbd>/</kbd> to jump to it.
        </p>
      )}
    </>
  );
}

async function Search({ searchParams }: { searchParams: SearchParams }) {
  const query = await readQuery(searchParams);
  if (query === "") {
    return null;
  }
  return (
    <Suspense fallback={<p className="text-muted">Searching…</p>}>
      <Results query={query} />
    </Suspense>
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
