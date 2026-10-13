"use client";

import { useEffect } from "react";

// Server errors arrive with a generic message and a digest that matches the
// server log; the API being down or timing out ends up here.
export default function ErrorPage({
  error,
  retry,
}: {
  error: Error & { digest?: string };
  retry: () => void;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold tracking-tight">
        Something went wrong
      </h1>
      <p className="text-zinc-600 dark:text-zinc-400">
        The page couldn&apos;t be loaded. Try again in a moment.
      </p>
      <button
        type="button"
        onClick={() => retry()}
        className="rounded border border-zinc-300 px-3 py-1.5 text-sm font-medium hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800"
      >
        Try again
      </button>
    </div>
  );
}
