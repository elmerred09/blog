// @vitest-environment node
import {
  getRedirectUrl,
  unstable_doesMiddlewareMatch,
} from "next/experimental/testing/server";
import { NextRequest } from "next/server";
import { describe, expect, it } from "vitest";

import nextConfig from "./next.config";
import { config, proxy } from "./proxy";

describe("proxy matcher", () => {
  it.each([
    ["/admin", true],
    ["/admin/posts/new", true],
    ["/", false],
    ["/posts/hello", false],
    ["/sign-in", false],
    ["/api/auth/callback/github", false],
  ])("%s runs the proxy: %s", (url, want) => {
    expect(unstable_doesMiddlewareMatch({ config, nextConfig, url })).toBe(
      want,
    );
  });
});

describe("proxy", () => {
  it("redirects a visitor with no session cookie to sign in", () => {
    const res = proxy(new NextRequest("http://localhost:3000/admin"));
    expect(res && getRedirectUrl(res)).toBe("http://localhost:3000/sign-in");
  });

  it.each([
    ["better-auth.session_token"],
    ["__Secure-better-auth.session_token"],
  ])("lets %s through to requireAdmin", (name) => {
    const req = new NextRequest("http://localhost:3000/admin", {
      headers: { cookie: `${name}=abc` },
    });
    expect(proxy(req)).toBeUndefined();
  });
});
