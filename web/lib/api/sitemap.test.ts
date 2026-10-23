// @vitest-environment node
import { cacheLife, cacheTag } from "next/cache";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { listSitemapEntries } from "@/lib/api";
import type { SitemapPage } from "@/lib/api";

vi.mock("server-only", () => ({}));
vi.mock("next/cache", () => ({ cacheLife: vi.fn(), cacheTag: vi.fn() }));

const API_URL = "http://api.test";

const page: SitemapPage = {
  items: [{ slug: "keyset-pagination", last_modified: "2026-10-02T08:30:00Z" }],
};

beforeEach(() => {
  vi.stubEnv("API_URL", API_URL);
});

describe("listSitemapEntries", () => {
  it("returns the entries", async () => {
    const requests: Request[] = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        requests.push(new Request(input, init));
        return new Response(JSON.stringify(page), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }),
    );

    await expect(listSitemapEntries()).resolves.toEqual(page.items);

    expect(requests).toHaveLength(1);
    expect(new URL(requests[0].url).pathname).toBe("/api/v1/sitemap");
    expect(cacheLife).toHaveBeenCalledExactlyOnceWith("hours");
    expect(cacheTag).toHaveBeenCalledExactlyOnceWith("posts");
  });
});
