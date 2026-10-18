import type { Metadata } from "next";
import { Suspense } from "react";

import { signInErrorMessage } from "@/lib/auth/messages";

import { signInWithGitHub } from "./actions";

export const metadata: Metadata = {
  title: "Sign in",
  robots: { index: false, follow: false },
};

export default function SignInPage({ searchParams }: PageProps<"/sign-in">) {
  return (
    <div className="mx-auto max-w-sm">
      <h1 className="mb-2 text-3xl font-bold tracking-tight">Sign in</h1>
      <p className="mb-6 text-zinc-600 dark:text-zinc-400">
        The editor is for the blog&apos;s author.
      </p>
      <Suspense fallback={null}>
        <SignInError searchParams={searchParams} />
      </Suspense>
      <form action={signInWithGitHub}>
        <button
          type="submit"
          className="rounded border border-zinc-300 px-3 py-1.5 text-sm font-medium hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800"
        >
          Sign in with GitHub
        </button>
      </form>
    </div>
  );
}

async function SignInError({
  searchParams,
}: {
  searchParams: PageProps<"/sign-in">["searchParams"];
}) {
  const { error } = await searchParams;
  if (typeof error !== "string" || error === "") {
    return null;
  }
  return (
    <p role="alert" className="mb-6 text-sm text-red-700 dark:text-red-400">
      {signInErrorMessage(error)}
    </p>
  );
}
