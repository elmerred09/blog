"use client";

import Form from "next/form";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useRef } from "react";

import { maxQueryLength, useSearchQuery } from "@/components/search-form";

type Props = {
  defaultQuery: string;
};

// The site-wide search bar. Below the sm breakpoint there's no room for it in
// the header, so it's an icon linking to /search, whose page shows its own
// form at that size.
export function HeaderSearch({ defaultQuery }: Props) {
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key !== "/" || event.metaKey || event.ctrlKey || event.altKey) {
        return;
      }
      if (isTypingTarget(event.target)) {
        return;
      }
      event.preventDefault();
      inputRef.current?.focus();
      inputRef.current?.select();
    }
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, []);

  return (
    <>
      <Form
        action="/search"
        role="search"
        className="relative hidden w-64 sm:block"
      >
        <SearchIcon className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-subtle" />
        <input
          ref={inputRef}
          type="search"
          name="q"
          aria-label="Search posts"
          aria-keyshortcuts="/"
          placeholder="Search posts"
          defaultValue={defaultQuery}
          maxLength={maxQueryLength}
          required
          className="peer h-10 w-full rounded-lg border border-line bg-card pr-9 pl-9 text-sm placeholder:text-subtle hover:border-line-strong"
        />
        <kbd
          aria-hidden="true"
          className="pointer-events-none absolute top-1/2 right-3 hidden -translate-y-1/2 rounded border border-line px-1.5 font-sans text-xs text-subtle peer-placeholder-shown:block peer-focus:hidden"
        >
          /
        </kbd>
      </Form>
      <Link
        href="/search"
        aria-label="Search"
        className="flex size-10 items-center justify-center rounded-lg border border-line bg-card hover:border-line-strong sm:hidden"
      >
        <SearchIcon className="size-4" />
      </Link>
    </>
  );
}

// HeaderSearchFromUrl shows the current query while on /search, and resets
// the input whenever that query changes or the reader leaves the page.
export function HeaderSearchFromUrl() {
  const pathname = usePathname();
  const query = useSearchQuery();
  const shown = pathname === "/search" ? query : "";
  return <HeaderSearch key={shown} defaultQuery={shown} />;
}

function isTypingTarget(target: EventTarget | null) {
  if (!(target instanceof HTMLElement)) {
    return false;
  }
  return (
    ["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName) ||
    target.closest("[contenteditable]") !== null
  );
}

function SearchIcon({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      aria-hidden="true"
      className={className}
    >
      <circle cx="11" cy="11" r="7" />
      <path d="m20 20-3.5-3.5" />
    </svg>
  );
}
