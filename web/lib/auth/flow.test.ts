// @vitest-environment node
import { betterAuth } from "better-auth";
import { getSessionCookie } from "better-auth/cookies";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { authOptions, readAuthEnv } from "./config";

// The real Better Auth sign-in, start to callback, with only GitHub's three
// endpoints stubbed. It checks what the plan relies on: the allowlist runs
// before a session exists, the session carries the GitHub id, and a signed-in
// user can't rewrite that id.

const BASE = "http://localhost:3000";
const ALLOWED = 123;

function newAuth() {
  const settings = readAuthEnv({
    BETTER_AUTH_SECRET: "test-secret-0123456789abcdef0123456789",
    BETTER_AUTH_URL: BASE,
    GITHUB_CLIENT_ID: "client",
    GITHUB_CLIENT_SECRET: "secret",
    ADMIN_GITHUB_IDS: String(ALLOWED),
  });
  return betterAuth({ ...authOptions(settings!), logger: { disabled: true } });
}

function stubGitHub(id: number) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: string | URL | Request) => {
      const url = input instanceof Request ? input.url : String(input);
      if (url.startsWith("https://github.com/login/oauth/access_token")) {
        return Response.json({
          access_token: "gho_test",
          token_type: "bearer",
          scope: "read:user,user:email",
        });
      }
      if (url === "https://api.github.com/user") {
        return Response.json({
          id,
          login: "elmer",
          name: "Elmer",
          email: null,
          avatar_url: "https://avatars.example/elmer",
        });
      }
      if (url === "https://api.github.com/user/emails") {
        return Response.json([
          { email: "elmer@example.com", primary: true, verified: true },
        ]);
      }
      throw new Error(`unexpected fetch ${url}`);
    }),
  );
}

// A minimal cookie jar: later Set-Cookie headers replace earlier ones, and
// an expired cookie is dropped.
class Jar {
  private cookies = new Map<string, string>();

  take(res: Response) {
    for (const line of res.headers.getSetCookie()) {
      const [pair, ...attrs] = line.split(";");
      const eq = pair.indexOf("=");
      const name = pair.slice(0, eq).trim();
      const expired = attrs.some((a) => /^\s*max-age=0\s*$/i.test(a));
      if (expired) {
        this.cookies.delete(name);
      } else {
        this.cookies.set(name, pair.slice(eq + 1));
      }
    }
  }

  header() {
    return [...this.cookies].map(([n, v]) => `${n}=${v}`).join("; ");
  }
}

async function signIn(auth: ReturnType<typeof newAuth>, jar: Jar) {
  const start = await auth.handler(
    new Request(`${BASE}/api/auth/sign-in/social`, {
      method: "POST",
      headers: { "content-type": "application/json", origin: BASE },
      body: JSON.stringify({
        provider: "github",
        callbackURL: "/admin",
        errorCallbackURL: "/sign-in",
      }),
    }),
  );
  expect(start.status).toBe(200);
  jar.take(start);
  const { url } = (await start.json()) as { url: string };
  const state = new URL(url).searchParams.get("state");

  const callback = await auth.handler(
    new Request(`${BASE}/api/auth/callback/github?code=abc&state=${state}`, {
      headers: { cookie: jar.header() },
    }),
  );
  jar.take(callback);
  return callback;
}

function postJSON(jar: Jar, path: string, body: unknown) {
  return new Request(`${BASE}/api/auth${path}`, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      origin: BASE,
      cookie: jar.header(),
    },
    body: JSON.stringify(body),
  });
}

describe("GitHub sign-in", () => {
  let auth: ReturnType<typeof newAuth>;
  let jar: Jar;

  beforeEach(() => {
    auth = newAuth();
    jar = new Jar();
  });

  it("gives an allowed account a session carrying its GitHub id", async () => {
    stubGitHub(ALLOWED);
    const callback = await signIn(auth, jar);

    expect(callback.status).toBe(302);
    expect(callback.headers.get("location")).toBe("/admin");
    const headers = new Headers({ cookie: jar.header() });
    expect(getSessionCookie(headers)).toBeTruthy();
    const session = await auth.api.getSession({ headers });
    expect(session?.user.githubId).toBe(String(ALLOWED));
  });

  it("rejects any other account before a session exists", async () => {
    stubGitHub(999);
    const callback = await signIn(auth, jar);

    expect(callback.status).toBe(302);
    const location = new URL(callback.headers.get("location")!, BASE);
    expect(location.pathname).toBe("/sign-in");
    expect(location.searchParams.get("error")).toBe("not_allowed");
    const headers = new Headers({ cookie: jar.header() });
    expect(getSessionCookie(headers)).toBeNull();
    expect(await auth.api.getSession({ headers })).toBeNull();
  });

  it("won't let a session change its own GitHub id", async () => {
    stubGitHub(ALLOWED);
    await signIn(auth, jar);

    const res = await auth.handler(
      postJSON(jar, "/update-user", { githubId: "999" }),
    );
    expect(res.ok).toBe(false);
    jar.take(res);

    const session = await auth.api.getSession({
      headers: new Headers({ cookie: jar.header() }),
    });
    expect(session?.user.githubId).toBe(String(ALLOWED));
  });

  it("still allows other profile updates", async () => {
    stubGitHub(ALLOWED);
    await signIn(auth, jar);

    const res = await auth.handler(
      postJSON(jar, "/update-user", { name: "Elmer R." }),
    );
    expect(res.status).toBe(200);
  });
});
