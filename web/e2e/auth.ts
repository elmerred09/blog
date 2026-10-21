import path from "node:path";

import type { BrowserContext } from "@playwright/test";
import { makeSignature, symmetricEncodeJWT } from "better-auth/crypto";

// Session cookies for an admin, built the way Better Auth 1.7.10 builds them
// in stateless mode, so the tests can sign in without the GitHub round trip.
// The format is Better Auth's, not ours: session_token is the token plus an
// HMAC of it (better-call's signed cookie), and session_data is the
// cookie-cache payload as a JWE (cookieCache.strategy "jwe" in
// lib/auth/config.ts). An upgrade that changes either shows up as admin
// specs redirecting to /sign-in.

// The names Better Auth uses when BETTER_AUTH_URL is http://. Under https
// they get a __Secure- prefix.
export const SESSION_TOKEN_COOKIE = "better-auth.session_token";
export const SESSION_DATA_COOKIE = "better-auth.session_data";

// The admin project's storageState, written by auth.setup.ts.
export const ADMIN_STORAGE_STATE = path.join(__dirname, ".auth", "admin.json");

// The name in the minted user, which the admin bar shows once signed in.
export const ADMIN_NAME = "E2E Admin";

export const VERSION_GUARD_MESSAGE =
  "a minted admin session didn't sign in; Better Auth's cookie format may have changed (better-auth upgrade?): update e2e/auth.ts";

export type SessionCookie = {
  name: string;
  value: string;
  // Unix seconds.
  expires: number;
};

export async function mintSessionCookies({
  secret,
  githubId,
  name = ADMIN_NAME,
  maxAgeSeconds = 60 * 60,
  now = new Date(),
}: {
  secret: string;
  githubId: string;
  name?: string;
  maxAgeSeconds?: number;
  now?: Date;
}): Promise<SessionCookie[]> {
  const expiresAt = new Date(now.getTime() + maxAgeSeconds * 1000);
  const token = crypto.randomUUID().replaceAll("-", "");
  const userId = `e2e-user-${githubId}`;

  const payload = {
    session: {
      id: `e2e-session-${token}`,
      token,
      userId,
      expiresAt,
      createdAt: now,
      updatedAt: now,
      ipAddress: "",
      userAgent: "",
    },
    user: {
      id: userId,
      name,
      email: `e2e-${githubId}@example.com`,
      emailVerified: true,
      image: null,
      createdAt: now,
      updatedAt: now,
      githubId,
    },
    updatedAt: now.getTime(),
    version: "1",
  };

  const signature = await makeSignature(token, secret);
  const expires = Math.floor(expiresAt.getTime() / 1000);
  return [
    {
      name: SESSION_TOKEN_COOKIE,
      value: encodeURIComponent(`${token}.${signature}`),
      expires,
    },
    {
      name: SESSION_DATA_COOKIE,
      value: await symmetricEncodeJWT(
        payload,
        secret,
        "better-auth-session",
        maxAgeSeconds,
      ),
      expires,
    },
  ];
}

// addSessionCookies puts cookies on context for baseURL's host, with the
// attributes Better Auth gives its own over http.
export async function addSessionCookies(
  context: BrowserContext,
  baseURL: string,
  cookies: SessionCookie[],
): Promise<void> {
  const domain = new URL(baseURL).hostname;
  await context.addCookies(
    cookies.map((cookie) => ({
      ...cookie,
      domain,
      path: "/",
      httpOnly: true,
      secure: false,
      sameSite: "Lax" as const,
    })),
  );
}
