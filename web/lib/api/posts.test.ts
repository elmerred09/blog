// @vitest-environment node
import { cacheLife, cacheTag } from "next/cache";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError, getPost, listPosts } from "@/lib/api";
import type { ErrorCode, Page, Post } from "@/lib/api";

vi.mock("server-only", () => ({}));
vi.mock("next/cache", () => ({ cacheLife: vi.fn(), cacheTag: vi.fn() }));

const API_URL = "http://api.test";

const page: Page = {
  items: [
    {
      slug: "keyset-pagination",
      title: "Keyset pagination",
      summary: "Why OFFSET gets slow.",
      published_at: "2026-10-01T12:00:00Z",
      tags: [{ slug: "postgres", name: "Postgres" }],
    },
  ],
  next_cursor: "MjAyNi0xMC0wMVQxMjowMDowMFp8YWJj",
};

const post: Post = {
  ...page.items[0],
  body_html: "<p>Hello.</p>",
  updated_at: "2026-10-02T08:30:00Z",
};

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function apiError(status: number, code: ErrorCode, message: string): Response {
  return json(status, { error: { code, message } });
}

// Replaces global fetch with one that answers every call with response, and
// returns the requests it received.
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

function only(requests: Request[]): URL {
  expect(requests).toHaveLength(1);
  expect(requests[0].method).toBe("GET");
  return new URL(requests[0].url);
}

beforeEach(() => {
  vi.stubEnv("API_URL", API_URL);
});

describe("listPosts", () => {
  it("requests the first page without a query string", async () => {
    const requests = stubFetch(json(200, page));

    await expect(listPosts()).resolves.toEqual(page);

    const url = only(requests);
    expect(url.origin + url.pathname).toBe(`${API_URL}/api/v1/posts`);
    expect(url.search).toBe("");
  });

  it("sends limit and cursor", async () => {
    const requests = stubFetch(json(200, page));

    await listPosts({ limit: 5, cursor: "abc_-1" });

    const { searchParams } = only(requests);
    expect(Object.fromEntries(searchParams)).toEqual({
      limit: "5",
      cursor: "abc_-1",
    });
  });

  it("keeps a null next_cursor on the last page", async () => {
    stubFetch(json(200, { ...page, next_cursor: null }));

    const result = await listPosts();

    expect(result.next_cursor).toBeNull();
  });

  it("is cached for minutes under the posts tag", async () => {
    stubFetch(json(200, page));

    await listPosts();

    expect(cacheLife).toHaveBeenCalledExactlyOnceWith("minutes");
    expect(new Set(vi.mocked(cacheTag).mock.calls.flat())).toEqual(
      new Set(["posts"]),
    );
  });
});

describe("getPost", () => {
  it("returns the post", async () => {
    const requests = stubFetch(json(200, post));

    await expect(getPost(post.slug)).resolves.toEqual(post);

    expect(only(requests).pathname).toBe(`/api/v1/posts/${post.slug}`);
  });

  it("encodes the slug as a single path segment", async () => {
    const requests = stubFetch(json(200, post));

    await getPost("../readyz");

    expect(only(requests).pathname).toBe("/api/v1/posts/..%2Freadyz");
  });

  it("returns null on 404", async () => {
    stubFetch(apiError(404, "not_found", "post not found"));

    await expect(getPost("missing")).resolves.toBeNull();
  });

  it("is cached for hours under the posts and post:<slug> tags", async () => {
    stubFetch(json(200, post));

    await getPost(post.slug);

    expect(cacheLife).toHaveBeenCalledExactlyOnceWith("hours");
    expect(new Set(vi.mocked(cacheTag).mock.calls.flat())).toEqual(
      new Set(["posts", `post:${post.slug}`]),
    );
  });
});

describe("error responses", () => {
  it.each([
    {
      name: "listPosts 400",
      call: () => listPosts({ cursor: "bad" }),
      response: apiError(400, "invalid_request", "invalid cursor"),
      want: { status: 400, code: "invalid_request", message: "invalid cursor" },
    },
    {
      name: "listPosts 500",
      call: () => listPosts(),
      response: apiError(500, "internal", "internal server error"),
      want: { status: 500, code: "internal", message: "internal server error" },
    },
    {
      name: "listPosts 503",
      call: () => listPosts(),
      response: apiError(503, "timeout", "request timed out"),
      want: { status: 503, code: "timeout", message: "request timed out" },
    },
    {
      name: "getPost 503",
      call: () => getPost(post.slug),
      response: apiError(503, "timeout", "request timed out"),
      want: { status: 503, code: "timeout", message: "request timed out" },
    },
    {
      name: "non-JSON body",
      call: () => listPosts(),
      response: new Response("<html>Bad Gateway</html>", {
        status: 502,
        headers: { "Content-Type": "text/html" },
      }),
      want: { status: 502, code: "unexpected_response" },
    },
    {
      name: "empty body",
      call: () => getPost(post.slug),
      response: new Response(null, {
        status: 502,
        headers: { "Content-Length": "0" },
      }),
      want: { status: 502, code: "unexpected_response" },
    },
  ])("$name throws ApiError", async ({ call, response, want }) => {
    stubFetch(response);

    const err = await call().catch((e: unknown) => e);

    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject(want);
  });

  it("lets a network failure through instead of turning it into null", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        throw new TypeError("fetch failed");
      }),
    );

    await expect(getPost(post.slug)).rejects.toThrow("fetch failed");
  });
});

describe("API_URL", () => {
  it("isn't needed to import the module", async () => {
    vi.stubEnv("API_URL", undefined);
    vi.resetModules();

    await expect(import("@/lib/api")).resolves.toHaveProperty("listPosts");
  });

  it("fails the call clearly when unset, without fetching", async () => {
    vi.stubEnv("API_URL", undefined);
    const requests = stubFetch(json(200, page));

    await expect(listPosts()).rejects.toThrow(/API_URL/);
    expect(requests).toHaveLength(0);
  });

  it("tolerates a trailing slash", async () => {
    vi.stubEnv("API_URL", `${API_URL}/`);
    const requests = stubFetch(json(200, page));

    await listPosts();

    expect(only(requests).pathname).toBe("/api/v1/posts");
  });
});
