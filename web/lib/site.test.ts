// @vitest-environment node
import { afterEach, describe, expect, it, vi } from "vitest";

import { indexingEnabled, siteUrl } from "@/lib/site";

afterEach(() => {
  vi.unstubAllEnvs();
});

describe("siteUrl", () => {
  it("returns SITE_URL without a trailing slash", () => {
    vi.stubEnv("SITE_URL", "https://blog.example/");
    expect(siteUrl()).toBe("https://blog.example");
  });

  it("falls back to localhost outside production", () => {
    vi.stubEnv("NODE_ENV", "test");
    vi.stubEnv("SITE_URL", "");
    expect(siteUrl()).toBe("http://localhost:3000");
  });

  it("throws in production when SITE_URL is unset", () => {
    vi.stubEnv("NODE_ENV", "production");
    vi.stubEnv("SITE_URL", "");
    expect(() => siteUrl()).toThrow(/SITE_URL/);
  });
});

describe("indexingEnabled", () => {
  it("is on only when SITE_INDEXING is true", () => {
    vi.stubEnv("SITE_INDEXING", "true");
    expect(indexingEnabled()).toBe(true);

    vi.stubEnv("SITE_INDEXING", "false");
    expect(indexingEnabled()).toBe(false);

    vi.stubEnv("SITE_INDEXING", "");
    expect(indexingEnabled()).toBe(false);
  });
});
