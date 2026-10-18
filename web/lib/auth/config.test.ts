// @vitest-environment node
import { describe, expect, it } from "vitest";

import { AUTH_ENV, allowlist, parseGithubIds, readAuthEnv } from "./config";

const full = {
  BETTER_AUTH_SECRET: "s".repeat(32),
  BETTER_AUTH_URL: "http://localhost:3000",
  GITHUB_CLIENT_ID: "client",
  GITHUB_CLIENT_SECRET: "secret",
  ADMIN_GITHUB_IDS: "123, 456",
};

describe("readAuthEnv", () => {
  it("is null when nothing is set", () => {
    expect(readAuthEnv({})).toBeNull();
  });

  it("treats empty values as unset", () => {
    const empty = Object.fromEntries(AUTH_ENV.map((name) => [name, ""]));
    expect(readAuthEnv(empty)).toBeNull();
  });

  it("names what's missing when partly set", () => {
    const partial = { ...full, GITHUB_CLIENT_ID: "", GITHUB_CLIENT_SECRET: "" };
    expect(() => readAuthEnv(partial)).toThrow(
      /set GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET/,
    );
  });

  it("reads a complete config", () => {
    const s = readAuthEnv(full);
    expect(s?.baseURL).toBe("http://localhost:3000");
    expect([...(s?.githubIds ?? [])]).toEqual(["123", "456"]);
  });
});

describe("parseGithubIds", () => {
  it.each([["1"], ["1,2"], [" 1 , 2 ,"]])("accepts %j", (raw) => {
    expect(parseGithubIds(raw).size).toBeGreaterThan(0);
  });

  it.each([["elmer"], ["1,elmer"], ["-1"], [" , "]])("rejects %j", (raw) => {
    expect(() => parseGithubIds(raw)).toThrow(/ADMIN_GITHUB_IDS/);
  });
});

describe("allowlist", () => {
  const check = allowlist(new Set(["123"]));
  const github = (profile?: Record<string, unknown>) => ({
    user: {},
    source: {
      action: "sign-in" as const,
      method: "oauth" as const,
      oauth: { providerId: "github", profile },
    },
  });
  // The hook doesn't use the endpoint context.
  const ctx = undefined as never;

  it("admits an allowed numeric id", async () => {
    expect(await check(github({ id: 123 }), ctx)).toBeUndefined();
  });

  it("admits an allowed id sent as a string", async () => {
    expect(await check(github({ id: "123" }), ctx)).toBeUndefined();
  });

  it.each([
    ["another id", github({ id: 456 })],
    ["no id", github({ login: "elmer" })],
    ["no profile", github()],
    [
      "another provider with an allowed id",
      {
        user: {},
        source: {
          action: "sign-in" as const,
          method: "oauth" as const,
          oauth: { providerId: "gitlab", profile: { id: 123 } },
        },
      },
    ],
    [
      "a non-OAuth method",
      {
        user: {},
        source: { action: "create-user" as const, method: "email-password" },
      },
    ],
  ])("rejects %s", async (_, data) => {
    expect(await check(data, ctx)).toMatchObject({ error: "not_allowed" });
  });
});
