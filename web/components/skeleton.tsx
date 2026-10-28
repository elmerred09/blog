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
