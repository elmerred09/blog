// The home page's opening: the statement is the page's h1, so it carries the
// weight and the bio stays quiet beneath it.
export function Intro() {
  return (
    <section className="mb-16 border-b border-foreground pb-14">
      <h1 className="text-[clamp(2rem,1.25rem+2vw,3.25rem)] leading-[1.1] font-semibold tracking-tight text-balance">
        Making sense of software,{" "}
        <span className="relative isolate inline-block whitespace-nowrap">
          <span
            aria-hidden="true"
            className="absolute -inset-x-1.5 bottom-[0.08em] -z-10 h-[0.42em] -rotate-2 rounded-[2px_12px_4px_8px] bg-brand/45"
          />
          bit by bit
        </span>
        .
      </h1>
      <div className="mt-10 space-y-4 text-lg leading-relaxed text-muted">
        <p>
          I&apos;m Elmer, a software engineer with six years of experience
          building and working with systems that need to remain fast, reliable,
          and understandable as they grow. I enjoy digging beneath the surface,
          understanding why things work the way they do, and exploring how
          engineering decisions shape the systems we build.
        </p>
        <p>
          <strong className="font-semibold text-foreground">
            Bits by Elmer
          </strong>{" "}
          is where I explore those ideas and share what I learn along the way. I
          write mostly about databases and system design, drawing from my work,
          my reading, and the questions that send me down a rabbit hole. My hope
          is to make complex ideas easier to understand and share insights that
          help other engineers build better software, one bit at a time.
        </p>
      </div>
    </section>
  );
}
