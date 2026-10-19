// @vitest-environment node
import {
  decodeProtectedHeader,
  exportPKCS8,
  generateKeyPair,
  jwtVerify,
  type CryptoKey,
} from "jose";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { signApiToken, TOKEN_TTL_SECONDS } from "./token";

vi.mock("server-only", () => ({}));

let publicKey: CryptoKey;
let privatePem: string;

beforeAll(async () => {
  const pair = await generateKeyPair("EdDSA", { extractable: true });
  publicKey = pair.publicKey;
  privatePem = await exportPKCS8(pair.privateKey);
});

beforeEach(() => {
  vi.stubEnv("ADMIN_JWT_PRIVATE_KEY", privatePem);
});

// The claims internal/auth requires: EdDSA, iss blog-web, aud blog-api, and an
// exp no more than five minutes out.
function verify(token: string) {
  return jwtVerify(token, publicKey, {
    algorithms: ["EdDSA"],
    issuer: "blog-web",
    audience: "blog-api",
  });
}

describe("signApiToken", () => {
  it("signs the claims the API verifies", async () => {
    const token = await signApiToken("123");

    const { payload } = await verify(token);
    expect(payload.sub).toBe("github:123");
    expect(decodeProtectedHeader(token)).toEqual({ alg: "EdDSA", typ: "JWT" });
  });

  it("expires shortly after it's issued", async () => {
    const now = new Date("2026-10-18T12:00:00.900Z");

    const token = await signApiToken("123", now);

    const payload = JSON.parse(
      Buffer.from(token.split(".")[1], "base64url").toString(),
    );
    const iat = Math.floor(now.getTime() / 1000);
    expect(payload.iat).toBe(iat);
    expect(payload.exp).toBe(iat + TOKEN_TTL_SECONDS);
    expect(TOKEN_TTL_SECONDS).toBeLessThanOrEqual(5 * 60);
  });

  it("accepts a key whose newlines are escaped", async () => {
    vi.stubEnv("ADMIN_JWT_PRIVATE_KEY", privatePem.replaceAll("\n", "\\n"));

    await expect(verify(await signApiToken("123"))).resolves.toBeDefined();
  });

  it("fails without a key", async () => {
    vi.stubEnv("ADMIN_JWT_PRIVATE_KEY", "");

    await expect(signApiToken("123")).rejects.toThrow(/ADMIN_JWT_PRIVATE_KEY/);
  });
});
