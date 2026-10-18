import "server-only";

import { betterAuth } from "better-auth";
import { nextCookies } from "better-auth/next-js";

import { authOptions, readAuthEnv, type AuthSettings } from "./config";

function create(settings: AuthSettings) {
  // nextCookies lets server actions (sign-in, sign-out) set cookies.
  return betterAuth({ ...authOptions(settings), plugins: [nextCookies()] });
}

export type Auth = ReturnType<typeof create>;

let configured: { auth: Auth; settings: AuthSettings } | null | undefined;

// getAuth builds Better Auth on first use, not at import: next build loads
// this module with NODE_ENV=production and no secret, and constructing it
// there throws. Null means admin sign-in isn't configured.
export function getAuth() {
  if (configured === undefined) {
    const settings = readAuthEnv(process.env);
    configured = settings && { auth: create(settings), settings };
  }
  return configured;
}
