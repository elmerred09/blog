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

export type SessionCookie = {
  name: string;
  value: string;
  // Unix seconds.
  expires: number;
};

export async function mintSessionCookies({
  secret,
  githubId,
  name = "E2E Admin",
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
