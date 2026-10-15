import type { SnippetFragment } from "@/lib/api";

type Props = {
  fragments: SnippetFragment[];
};

// Fragments are plain text from the post's markdown source, so they're
// rendered as text, never as HTML.
export function Snippet({ fragments }: Props) {
  if (fragments.length === 0) {
    return null;
  }
  return (
    <p className="text-zinc-700 dark:text-zinc-300">
      {fragments.map((f, i) =>
        f.match ? (
          <mark
            key={i}
            className="rounded-sm bg-amber-200 px-0.5 text-inherit dark:bg-amber-700/60"
          >
            {f.text}
          </mark>
        ) : (
          <span key={i}>{f.text}</span>
        ),
      )}
    </p>
  );
}
