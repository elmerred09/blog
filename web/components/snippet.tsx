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
    <p className="text-sm leading-relaxed text-muted">
      {fragments.map((f, i) =>
        f.match ? (
          <mark
            key={i}
            className="rounded-sm bg-chip px-0.5 font-medium text-brand-strong"
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
