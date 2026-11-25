import Link from "next/link";

export function BackLink() {
  return (
    <Link
      href="/admin"
      className="mb-6 inline-flex items-center gap-1.5 text-sm font-medium text-muted hover:text-foreground"
    >
      <svg
        viewBox="0 0 16 16"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
        className="size-3.5"
      >
        <path d="M13 8H3M7 4 3 8l4 4" />
      </svg>
      Back
    </Link>
  );
}
