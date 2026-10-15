// @vitest-environment node
import { cacheLife, cacheTag } from "next/cache";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError, searchPosts } from "@/lib/api";
import type { SearchResults } from "@/lib/api";

vi.mock("server-only", () => ({}));
vi.mock("next/cache", () => ({ cacheLife: vi.fn(), cacheTag: vi.fn() }));

const API_URL = "http://api.test";

const results: SearchResults = {
  items: [
    {
      slug: "keyset-pagination",
      title: "Keyset pagination",
      summary: "Why OFFSET gets slow.",
      published_at: "2026-10-01T12:00:00Z",
      tags: [{ slug: "postgres", name: "Postgres" }],
      snippet: [
        { text: "use ", match: false },
        { text: "keyset", match: true },
        { text: " pagination", match: false },
      ],
    },
  ],
};

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function stubFetch(response: Response): Request[] {
  const requests: Request[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      requests.push(new Request(input, init));
      return response.clone();
    }),
  );
  return requests;
}

beforeEach(() => {
  vi.stubEnv("API_URL", API_URL);
});

describe("searchPosts", () => {
  it("sends q and returns the results", async () => {
    const requests = stubFetch(json(200, results));

    await expect(searchPosts({ q: "keyset" })).resolves.toEqual(results);

    expect(requests).toHaveLength(1);
    const url = new URL(requests[0].url);
    expect(url.pathname).toBe("/api/v1/search");
    expect(Object.fromEntries(url.searchParams)).toEqual({ q: "keyset" });
  });

  it("sends limit, and encodes q", async () => {
    const requests = stubFetch(json(200, results));

    await searchPosts({ q: '"keyset pagination" -offset & co', limit: 5 });

    const { searchParams } = new URL(requests[0].url);
    expect(searchParams.get("q")).toBe('"keyset pagination" -offset & co');
    expect(searchParams.get("limit")).toBe("5");
  });

  it("returns null when the API rejects the query", async () => {
    stubFetch(
      json(400, {
        error: { code: "invalid_request", message: "q is required" },
      }),
    );

    await expect(searchPosts({ q: " " })).resolves.toBeNull();
  });

  it("throws ApiError on a server error", async () => {
    stubFetch(
      json(503, { error: { code: "timeout", message: "request timed out" } }),
    );

    const err = await searchPosts({ q: "go" }).catch((e: unknown) => e);

    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({ status: 503, code: "timeout" });
  });

  it("is cached for minutes under the posts tag", async () => {
    stubFetch(json(200, results));

    await searchPosts({ q: "go" });

    expect(cacheLife).toHaveBeenCalledExactlyOnceWith("minutes");
    expect(new Set(vi.mocked(cacheTag).mock.calls.flat())).toEqual(
      new Set(["posts"]),
    );
  });
});
