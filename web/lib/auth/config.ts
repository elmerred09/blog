import { APIError } from "better-auth/api";
import type { BetterAuthOptions } from "better-auth";

// Admin sign-in is configured all or nothing, like the Go API's admin
// routes: with none of these set, /admin and /api/auth are 404s; with only
// some, the first admin request fails with an error naming the missing ones.
export const AUTH_ENV = [
  "BETTER_AUTH_SECRET",
  "BETTER_AUTH_URL",
  "GITHUB_CLIENT_ID",
  "GITHUB_CLIENT_SECRET",
  "ADMIN_GITHUB_IDS",
  // Read by lib/api/token.ts, which signs each admin API call. Listed here so
  // a signed-in admin can't hit a missing key on the first save.
  "ADMIN_JWT_PRIVATE_KEY",
] as const;

type Env = Record<string, string | undefined>;

export type AuthSettings = {
  secret: string;
  baseURL: string;
  github: { clientId: string; clientSecret: string };
  // Numeric GitHub user ids, as strings. Logins can be renamed; ids can't.
  githubIds: ReadonlySet<string>;
};

export function readAuthEnv(env: Env): AuthSettings | null {
  const missing = AUTH_ENV.filter((name) => !env[name]);
  if (missing.length === AUTH_ENV.length) {
    return null;
  }
  if (missing.length > 0) {
    throw new Error(
      `admin sign-in is partly configured; set ${missing.join(", ")} (or none of ${AUTH_ENV.join(", ")})`,
    );
  }
  return {
    secret: env.BETTER_AUTH_SECRET!,
    baseURL: env.BETTER_AUTH_URL!,
    github: {
      clientId: env.GITHUB_CLIENT_ID!,
      clientSecret: env.GITHUB_CLIENT_SECRET!,
    },
    githubIds: parseGithubIds(env.ADMIN_GITHUB_IDS!),
  };
}

// parseGithubIds reads a comma-separated list of numeric GitHub user ids.
export function parseGithubIds(raw: string): ReadonlySet<string> {
  const ids = raw
    .split(",")
    .map((id) => id.trim())
    .filter((id) => id !== "");
  const bad = ids.filter((id) => !/^\d+$/.test(id));
  if (bad.length > 0 || ids.length === 0) {
    throw new Error(
      `ADMIN_GITHUB_IDS must be numeric GitHub user ids, comma-separated; got ${JSON.stringify(raw)}`,
    );
  }
  return new Set(ids);
}

type ValidateUserInfo = NonNullable<
  NonNullable<BetterAuthOptions["user"]>["validateUserInfo"]
>;

// allowlist admits a GitHub sign-in only when the raw profile's numeric id is
// in ids. Better Auth runs it before creating the user and on every later
// GitHub sign-in, so a rejected account never gets a session.
export function allowlist(ids: ReadonlySet<string>): ValidateUserInfo {
  return ({ source }) => {
    const id =
      source.method === "oauth" && source.oauth?.providerId === "github"
        ? source.oauth.profile?.id
        : undefined;
    if (
      (typeof id === "number" || typeof id === "string") &&
      ids.has(String(id))
    ) {
      return;
    }
    return {
      error: "not_allowed",
      errorDescription: "This GitHub account can't sign in here.",
    };
  };
}

export function authOptions(s: AuthSettings) {
  return {
    secret: s.secret,
    baseURL: s.baseURL,
    // No database: the session is the encrypted cookie below, and nothing
    // about users or accounts outlives it. Better Auth defaults to these
    // values without a database; they're spelled out so a change is visible.
    session: {
      cookieCache: {
        enabled: true,
        strategy: "jwe",
        refreshCache: true,
        maxAge: 60 * 60 * 24 * 7,
      },
    },
    socialProviders: {
      github: {
        clientId: s.github.clientId,
        clientSecret: s.github.clientSecret,
        // The id the API token's sub is built from. It comes from the
        // provider-verified profile, never from the client.
        mapProfileToUser: (profile: { id: string | number }) => ({
          githubId: String(profile.id),
        }),
      },
    },
    user: {
      additionalFields: {
        // input stays true because Better Auth drops input: false fields
        // from mapProfileToUser when it creates the user. The update hook
        // below is what keeps it read-only.
        githubId: { type: "string", required: false },
      },
      validateUserInfo: allowlist(s.githubIds),
    },
    databaseHooks: {
      user: {
        update: {
          // Every user write goes through here, including the public
          // update-user endpoint, so no request can change its GitHub id.
          before: async (user: Record<string, unknown>) => {
            if ("githubId" in user) {
              throw new APIError("BAD_REQUEST", {
                message: "githubId can't be changed",
              });
            }
          },
        },
      },
    },
    onAPIError: { errorURL: "/sign-in" },
  } satisfies BetterAuthOptions;
}
