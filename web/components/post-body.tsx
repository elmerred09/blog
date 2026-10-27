import "@/app/chroma.css";
import "@/app/chroma-light.css";

type Props = {
  html: string;
};

// PostBody is shared by the public post page and the editor's preview, so a
// preview looks exactly like the published page.
export function PostBody({ html }: Props) {
  return (
    // html comes from our markdown renderer, which escapes raw HTML.
    <div
      className={[
        "prose max-w-none prose-pre:text-inherit",
        // Colors and sizes follow the devTalk template's post page.
        "[--tw-prose-body:rgba(0,0,0,0.8)] [--tw-prose-bold:#000] [--tw-prose-headings:#000]",
        "[--tw-prose-links:var(--brand)] [--tw-prose-quote-borders:var(--brand)]",
        "[--tw-prose-bullets:rgba(0,0,0,0.4)] [--tw-prose-counters:rgba(0,0,0,0.6)]",
        "prose-p:leading-7 prose-li:leading-7",
        "prose-h2:mt-10 prose-h2:mb-4 prose-h2:text-[2rem] prose-h2:leading-tight prose-h2:font-bold",
        "prose-h3:mt-8 prose-h3:mb-4 prose-h3:text-2xl prose-h3:font-bold",
        "prose-a:font-normal prose-a:no-underline hover:prose-a:underline",
        "prose-blockquote:border-l-8 prose-blockquote:text-xl prose-blockquote:font-normal prose-blockquote:not-italic",
      ].join(" ")}
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}
