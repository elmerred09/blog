"use client";

import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";

type Props = {
  html: string;
  className: string;
};

type CodeBlock = { host: HTMLElement; pre: HTMLPreElement };

// Rendered post HTML, with a copy button on each code block. The buttons are
// added after mount, into the HTML React doesn't manage, so the server HTML
// and hydration stay the plain rendered post.
export function RenderedHtml({ html, className }: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const [blocks, setBlocks] = useState<CodeBlock[]>([]);

  useEffect(() => {
    const root = ref.current;
    if (root === null) {
      return;
    }
    setBlocks(Array.from(root.querySelectorAll("pre"), withButtonHost));
  }, [html]);

  return (
    <>
      {/* html comes from our markdown renderer, which escapes raw HTML. */}
      <div
        ref={ref}
        className={className}
        dangerouslySetInnerHTML={{ __html: html }}
      />
      {blocks.map(({ host, pre }, i) =>
        createPortal(<CopyButton pre={pre} />, host, i),
      )}
    </>
  );
}

// Wraps the pre in a positioned div with a host for the button beside it.
// Outside the pre, the button stays put when the code scrolls sideways. The
// wrapper takes over the pre's margins as they were computed in place, so
// prose rules that depend on position (no top margin after a heading) still
// hold and nothing moves. Idempotent: an effect can run twice in dev.
function withButtonHost(pre: HTMLPreElement): CodeBlock {
  let wrapper = pre.parentElement;
  if (wrapper === null || !("codeBlock" in wrapper.dataset)) {
    const { marginTop, marginBottom } = getComputedStyle(pre);
    wrapper = document.createElement("div");
    wrapper.dataset.codeBlock = "";
    wrapper.className = "group/code relative";
    wrapper.style.marginTop = marginTop;
    wrapper.style.marginBottom = marginBottom;
    pre.style.marginTop = "0";
    pre.style.marginBottom = "0";
    pre.before(wrapper);
    wrapper.append(pre);
  }
  let host = wrapper.querySelector<HTMLElement>(":scope > [data-copy-host]");
  if (host === null) {
    host = document.createElement("div");
    host.dataset.copyHost = "";
    host.className = "absolute top-2 right-2";
    wrapper.append(host);
  }
  return { host, pre };
}

type CopyState = "idle" | "copied" | "failed";

const labels: Record<CopyState, string> = {
  idle: "Copy",
  copied: "Copied",
  failed: "Copy failed",
};

function CopyButton({ pre }: { pre: HTMLPreElement }) {
  const [state, setState] = useState<CopyState>("idle");

  useEffect(() => {
    if (state === "idle") {
      return;
    }
    const timer = setTimeout(() => setState("idle"), 2000);
    return () => clearTimeout(timer);
  }, [state]);

  async function copy() {
    // Each line ends in a newline, including the last.
    const text = (pre.textContent ?? "").replace(/\n$/, "");
    try {
      await navigator.clipboard.writeText(text);
      setState("copied");
    } catch {
      setState("failed");
    }
  }

  return (
    <button
      type="button"
      aria-label={state === "idle" ? "Copy code" : labels[state]}
      onClick={copy}
      className={[
        "rounded-md border border-line bg-background px-2 py-1 font-sans text-xs font-medium text-muted",
        "hover:border-line-strong hover:text-foreground",
        // Idle, it shows on hover or focus where there's a pointer to hover
        // with, and always on touch screens. After a click it stays to report.
        state === "idle"
          ? "group-hover/code:opacity-100 focus-visible:opacity-100 [@media(hover:hover)]:opacity-0"
          : "",
      ].join(" ")}
    >
      {labels[state]}
    </button>
  );
}
