import "server-only";

import { headers } from "next/headers";
import { notFound, redirect } from "next/navigation";
import { cache } from "react";

import { getAuth } from "@/lib/auth";

export type Admin = {
  githubId: string;
  name: string;
};

// requireAdmin is the one place that reads the session. Every admin page and
// every server action calls it; proxy.ts only skips the obvious cases. It
// re-checks the GitHub id against ADMIN_GITHUB_IDS, so removing an id locks
// that account out at the next restart, without waiting for its cookie to
// expire. With admin sign-in unconfigured, admin pages are 404s.
export const requireAdmin = cache(async (): Promise<Admin> => {
  // Read the request first. Checking the config first would let next build,
  // where admin env is usually unset, prerender a permanent 404.
  const requestHeaders = await headers();
  const configured = getAuth();
  if (!configured) {
    notFound();
  }
  const session = await configured.auth.api.getSession({
    headers: requestHeaders,
  });
  const githubId = session?.user.githubId;
  if (!session || !githubId) {
    redirect("/sign-in");
  }
  if (!configured.settings.githubIds.has(githubId)) {
    redirect("/sign-in?error=not_allowed");
  }
  return { githubId, name: session.user.name };
});
