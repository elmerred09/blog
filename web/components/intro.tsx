import { SocialLinks } from "@/components/social-links";

// The home page's opening: the statement is the page's h1, so it carries the
// weight and the bio stays quiet beneath it.
export function Intro() {
  return (
    <section className="mb-16 border-b border-line pb-14">
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
      <div className="mt-10 max-w-[65ch] space-y-4 text-lg leading-relaxed text-muted">
        <p>
          I&apos;m Elmer, a software engineer with six years of experience
          building and working with systems that need to remain fast, reliable,
          and easy to reason about as they grow. I enjoy finding out why things
          behave the way they do, and how the engineering decisions behind them
          hold up over time.
        </p>
        <p>
          This is where I explore those ideas and share what I learn along the
          way. I write mostly about databases and system design, drawing from my
          day job, my reading, and the questions that send me down a rabbit
          hole. My hope is to make complex ideas easier to grasp and give other
          engineers something useful to take back to their own systems.
        </p>
      </div>
      <SocialLinks className="mt-8" />
    </section>
  );
}
