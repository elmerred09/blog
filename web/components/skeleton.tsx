// A light streak crossing the whole placeholder, so the bars flash together.
// Hidden when the user prefers reduced motion; the gray shapes stay. The
// parent clips it, which is the card's own radius for a post.
function Shimmer() {
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none absolute inset-0 animate-shimmer bg-[linear-gradient(90deg,transparent_30%,rgba(255,255,255,0.9)_50%,transparent_70%)] motion-reduce:hidden"
    />
  );
}

const topicChipWidths = ["w-20", "w-16", "w-24", "w-14", "w-28"];

// Stand-in for TopicList. The heading is real: it does not come from the API.
export function TopicListSkeleton() {
  return (
    <section aria-labelledby="topics-heading" aria-busy="true">
      <h2
        id="topics-heading"
        className="mb-6 text-2xl font-semibold tracking-tight"
      >
        Dive in by topic
      </h2>
      <p className="sr-only" role="status">
        Loading topics…
      </p>
      <div className="relative overflow-hidden">
        <ul aria-hidden="true" className="flex flex-wrap gap-1">
          {topicChipWidths.map((width) => (
            <li key={width}>
              <div className={`h-10 rounded-lg bg-line ${width}`} />
            </li>
          ))}
        </ul>
        <Shimmer />
      </div>
    </section>
  );
}

const postCards = [
  { title: "w-2/3", summary: "w-4/5" },
  { title: "w-1/2", summary: "w-3/5" },
  { title: "w-3/4", summary: "w-5/6" },
];

// Stand-in for the home post list: the same card bands, with no text.
export function PostListSkeleton() {
  return (
    <div aria-busy="true">
      <p className="sr-only" role="status">
        Loading posts…
      </p>
      <ul aria-hidden="true" className="space-y-5">
        {postCards.map((card) => (
          <li key={card.title}>
            <article className="relative overflow-hidden rounded-2xl border border-line bg-card">
              <div className="border-b border-line px-6 py-4">
                <div className="h-3 w-24 rounded-sm bg-line" />
                <div className={`mt-2 h-5 rounded-sm bg-line ${card.title}`} />
              </div>
              <div className="space-y-4 px-6 py-5">
                <div className="space-y-2">
                  <div className="h-3.5 w-full rounded-sm bg-line" />
                  <div className={`h-3.5 rounded-sm bg-line ${card.summary}`} />
                </div>
                <div className="h-3.5 w-24 rounded-sm bg-line" />
              </div>
              <div className="flex gap-2 border-t border-line px-6 py-4">
                <div className="h-5 w-16 rounded-md bg-line" />
                <div className="h-5 w-14 rounded-md bg-line" />
              </div>
              <Shimmer />
            </article>
          </li>
        ))}
      </ul>
    </div>
  );
}

// Stand-in for the tag page's heading. The "Topic" label is real; the
// tag's name comes from the API.
export function TagHeadingSkeleton() {
  return (
    <header className="mb-10 space-y-2">
      <p className="text-sm font-medium text-muted">Topic</p>
      <div className="relative overflow-hidden">
        <div aria-hidden="true" className="h-12 w-64 rounded-sm bg-line" />
        <Shimmer />
      </div>
    </header>
  );
}

const postBodyLines = [
  ["full", "w-full"],
  ["full-2", "w-full"],
  ["long", "w-11/12"],
  ["shorter", "w-4/5"],
] as const;

// Stand-in for the post page: the date, title, and tags, then a short article.
export function PostSkeleton() {
  return (
    <article aria-busy="true" className="relative overflow-hidden">
      <p className="sr-only" role="status">
        Loading post…
      </p>
      <header className="mb-12 space-y-4">
        <div aria-hidden="true" className="h-3 w-24 rounded-sm bg-line" />
        <div aria-hidden="true" className="h-12 w-3/5 rounded-sm bg-line" />
        <div aria-hidden="true" className="flex gap-2 pt-2">
          <div className="h-6 w-16 rounded-md bg-line" />
          <div className="h-6 w-20 rounded-md bg-line" />
        </div>
      </header>
      <div aria-hidden="true" className="space-y-3">
        {postBodyLines.map(([key, width]) => (
          <div key={key} className={`h-4 rounded-sm bg-line ${width}`} />
        ))}
      </div>
      <div
        aria-hidden="true"
        className="mt-10 mb-4 h-8 w-1/2 rounded-sm bg-line"
      />
      <div aria-hidden="true" className="space-y-3">
        <div className="h-4 w-full rounded-sm bg-line" />
        <div className="h-4 w-full rounded-sm bg-line" />
        <div className="h-4 w-2/3 rounded-sm bg-line" />
      </div>
      <Shimmer />
    </article>
  );
}

const adminRows = [
  { title: "w-2/3", badge: "w-14" },
  { title: "w-1/2", badge: "w-20" },
  { title: "w-3/4", badge: "w-16" },
  { title: "w-2/5", badge: "w-14" },
  { title: "w-3/5", badge: "w-20" },
  { title: "w-4/5", badge: "w-16" },
];

// Stand-in for the admin post table. The column names are real: they do not
// come from the API. A handful of rows, enough to fill the first screen.
export function AdminPostListSkeleton() {
  return (
    <div
      aria-busy="true"
      className="max-lg:flex max-lg:min-h-0 max-lg:flex-1 max-lg:flex-col"
    >
      <p className="sr-only" role="status">
        Loading posts…
      </p>
      <div className="overflow-x-auto max-lg:min-h-0 max-lg:flex-1 max-lg:overflow-y-auto max-lg:overscroll-contain">
        <div className="relative min-w-3xl overflow-hidden rounded-2xl border border-line">
          <table className="w-full table-fixed text-left text-sm">
            <thead className="border-b border-line bg-card text-muted">
              <tr>
                <th className="px-6 py-3 font-medium">Title</th>
                <th className="w-32 px-4 py-3 font-medium">Status</th>
                <th className="w-40 px-4 py-3 font-medium">Published</th>
                <th className="w-40 px-4 py-3 font-medium">Updated</th>
                <th className="w-36 px-6 py-3">
                  <span className="sr-only">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody aria-hidden="true" className="divide-y divide-line">
              {adminRows.map((row) => (
                <tr key={row.title}>
                  <td className="px-6 py-4">
                    <div className={`h-4 rounded-sm bg-line ${row.title}`} />
                  </td>
                  <td className="px-4 py-4">
                    <div className={`h-5 rounded-full bg-line ${row.badge}`} />
                  </td>
                  <td className="px-4 py-4">
                    <div className="h-3.5 w-28 rounded-sm bg-line" />
                  </td>
                  <td className="px-4 py-4">
                    <div className="h-3.5 w-28 rounded-sm bg-line" />
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex justify-end gap-2">
                      <div className="size-10 rounded-lg border border-line" />
                      <div className="size-10 rounded-lg border border-line" />
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <Shimmer />
        </div>
      </div>
    </div>
  );
}

const toolbarMarks = [
  ["bold", "w-6"],
  ["italic", "w-6"],
  ["strike", "w-6"],
  ["code", "w-8"],
  ["h2", "w-8"],
  ["h3", "w-8"],
  ["list", "w-6"],
  ["link", "w-10"],
  ["block", "w-12"],
  ["image", "w-14"],
] as const;

// Stand-in for the editor form, shared by the new and edit pages. The labels
// are real: they do not come from the API.
export function PostEditorSkeleton() {
  return (
    <div aria-busy="true" className="relative overflow-hidden">
      <p className="sr-only" role="status">
        Loading editor…
      </p>
      <div aria-hidden="true" className="space-y-6">
        <EditorField label="Title" />
        <EditorField label="Slug" />
        <EditorField label="Summary" />
        <div className="space-y-2">
          <div className="flex gap-4 text-sm font-medium">
            <span className="border-b-2 border-foreground pb-1">Write</span>
            <span className="pb-1 text-muted">Preview</span>
          </div>
          <div className="rounded border border-zinc-300">
            <div className="flex min-h-10 flex-wrap items-center gap-1 border-b border-zinc-200 px-2 py-1">
              {toolbarMarks.map(([key, width]) => (
                <div key={key} className={`h-6 rounded-sm bg-line ${width}`} />
              ))}
            </div>
            <div className="min-h-96 space-y-3 px-4 py-3">
              <div className="h-4 w-full rounded-sm bg-line" />
              <div className="h-4 w-full rounded-sm bg-line" />
              <div className="h-4 w-11/12 rounded-sm bg-line" />
              <div className="h-4 w-4/5 rounded-sm bg-line" />
            </div>
          </div>
        </div>
        <EditorField label="Tags" />
        <div className="space-y-2">
          <p className="text-sm font-medium">Publishing</p>
          <div className="flex flex-wrap gap-x-5 gap-y-1">
            {["w-12", "w-24", "w-16"].map((width) => (
              <div key={width} className="flex items-center gap-1.5">
                <div className="size-4 rounded-full bg-line" />
                <div className={`h-3.5 rounded-sm bg-line ${width}`} />
              </div>
            ))}
          </div>
        </div>
        <div className="h-9 w-16 rounded bg-line" />
      </div>
      <Shimmer />
    </div>
  );
}

function EditorField({ label }: { label: string }) {
  return (
    <div className="space-y-1">
      <p className="text-sm font-medium">{label}</p>
      <div className="h-10 rounded border border-zinc-300 bg-card" />
    </div>
  );
}
