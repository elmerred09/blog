import "@/app/chroma.css";

type Props = {
  html: string;
};

// PostBody is shared by the public post page and the editor's preview, so a
// preview looks exactly like the published page.
export function PostBody({ html }: Props) {
  return (
    // html comes from our markdown renderer, which escapes raw HTML.
    <div
      className="prose max-w-none prose-zinc dark:prose-invert prose-pre:text-inherit"
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}
