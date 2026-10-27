// @vitest-environment node
import { cacheLife, cacheTag } from "next/cache";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError, listTags } from "@/lib/api";
import type { Tag } from "@/lib/api";

vi.mock("server-only", () => ({}));
vi.mock("next/cache", () => ({ cacheLife: vi.fn(), cacheTag: vi.fn() }));

const API_URL = "http://api.test";

const tags: Tag[] = [
  { slug: "go", name: "Go" },
  { slug: "postgres", name: "Postgres" },
];

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

describe("listTags", () => {
  it("returns the tags from /api/v1/tags", async () => {
    const requests = stubFetch(json(200, { tags }));

    await expect(listTags()).resolves.toEqual(tags);

    const url = new URL(requests[0].url);
    expect(requests[0].method).toBe("GET");
    expect(url.origin + url.pathname).toBe(`${API_URL}/api/v1/tags`);
    expect(url.search).toBe("");
  });

  it("returns an empty list", async () => {
    stubFetch(json(200, { tags: [] }));

    await expect(listTags()).resolves.toEqual([]);
  });

  it("is cached for minutes under the posts tag", async () => {
    stubFetch(json(200, { tags }));

    await listTags();

    expect(cacheLife).toHaveBeenCalledExactlyOnceWith("minutes");
    expect(new Set(vi.mocked(cacheTag).mock.calls.flat())).toEqual(
      new Set(["posts"]),
    );
  });

  it("throws ApiError when the API fails", async () => {
    stubFetch(
      json(500, {
        error: { code: "internal", message: "internal server error" },
      }),
    );

    await expect(listTags()).rejects.toBeInstanceOf(ApiError);
  });
});
