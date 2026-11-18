"use client";

import Form from "next/form";
import { useSearchParams } from "next/navigation";

// The API's limit; it checks the trimmed text, so this only spares a request.
export const maxQueryLength = 200;

// useSearchQuery is the trimmed ?q= of the current URL.
export function useSearchQuery() {
  return useSearchParams().get("q")?.trim() ?? "";
}

type Props = {
  defaultQuery: string;
};

export function SearchForm({ defaultQuery }: Props) {
  return (
    <Form action="/search" role="search" className="flex gap-2">
      <input
        type="search"
        name="q"
        aria-label="Search posts"
        placeholder="Search posts"
        defaultValue={defaultQuery}
        maxLength={maxQueryLength}
        required
        className="min-w-0 flex-1 rounded-lg border border-line bg-card px-5 py-2.5 text-sm placeholder:text-subtle hover:border-line-strong"
      />
      <button
        type="submit"
        className="rounded-lg border border-line bg-card px-5 py-2.5 text-sm font-medium hover:border-line-strong"
      >
        Search
      </button>
    </Form>
  );
}

export function SearchFormFromUrl() {
  const query = useSearchQuery();
  // The key resets the input when back/forward changes the query.
  return <SearchForm key={query} defaultQuery={query} />;
}
