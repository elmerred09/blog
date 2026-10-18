"use server";

import { headers } from "next/headers";
import { redirect } from "next/navigation";

import { getAuth } from "@/lib/auth";

// Starts the GitHub OAuth flow. Better Auth sets the state cookie (through
// nextCookies) and returns GitHub's authorization URL. The allowlist runs in
// the callback, before any session exists.
export async function signInWithGitHub() {
  const configured = getAuth();
  if (!configured) {
    redirect("/sign-in?error=not_configured");
  }
  const { url } = await configured.auth.api.signInSocial({
    body: {
      provider: "github",
      callbackURL: "/admin",
      errorCallbackURL: "/sign-in",
    },
    headers: await headers(),
  });
  if (!url) {
    throw new Error("sign-in: Better Auth returned no GitHub URL");
  }
  redirect(url);
}

export async function signOut() {
  const configured = getAuth();
  if (configured) {
    await configured.auth.api.signOut({ headers: await headers() });
  }
  redirect("/sign-in");
}
