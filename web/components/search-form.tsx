import Form from "next/form";

// The API's limit; it checks the trimmed text, so this only spares a request.
const maxQueryLength = 200;

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
        className="min-w-0 flex-1 rounded border border-zinc-300 bg-transparent px-3 py-1.5 dark:border-zinc-700"
      />
      <button
        type="submit"
        className="rounded border border-zinc-300 px-3 py-1.5 text-sm font-medium hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800"
      >
        Search
      </button>
    </Form>
  );
}
