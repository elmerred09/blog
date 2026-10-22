// @vitest-environment node
import { exportPKCS8, generateKeyPair, jwtVerify, type CryptoKey } from "jose";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import type { Admin } from "@/lib/admin/session";

import {
  createAdminPost,
  createImageUpload,
  getAdminPost,
  listAdminPosts,
  listAdminTags,
  previewPost,
  updateAdminPost,
} from "./admin";
import { ApiError } from "./errors";
import type {
  AdminPage,
  AdminPost,
  AdminPostInput,
  ErrorCode,
  Problem,
  Upload,
} from "./types";

vi.mock("server-only", () => ({}));

const API_URL = "http://api.test";
const admin: Admin = { githubId: "123", name: "Elmer" };

// Microseconds, as Postgres stores them. A Date would truncate to
// milliseconds and every save would be a false conflict.
const LOCK = "2026-10-18T12:00:12.345678Z";

const post: AdminPost = {
  id: "0f8a5c1e-4b2d-4c3a-9e1f-2a3b4c5d6e7f",
  slug: "keyset-pagination",
  title: "Keyset pagination",
  summary: "Why OFFSET gets slow.",
  body_md: "Hello.",
  published_at: null,
  created_at: "2026-10-18T11:00:00Z",
  updated_at: LOCK,
  state: "draft",
  tags: [{ slug: "postgres", name: "Postgres" }],
};

const input: AdminPostInput = {
  slug: post.slug,
  title: post.title,
  summary: post.summary,
  body_md: post.body_md,
  published_at: null,
  tags: ["Postgres"],
};

let publicKey: CryptoKey;
let privatePem: string;

beforeAll(async () => {
  const pair = await generateKeyPair("EdDSA", { extractable: true });
  publicKey = pair.publicKey;
  privatePem = await exportPKCS8(pair.privateKey);
});

beforeEach(() => {
  vi.stubEnv("API_URL", API_URL);
  vi.stubEnv("ADMIN_JWT_PRIVATE_KEY", privatePem);
});

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function apiError(
  status: number,
  code: ErrorCode,
  message: string,
  problems?: Problem[],
): Response {
  return json(status, { error: { code, message, problems } });
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

function only(requests: Request[]): Request {
  expect(requests).toHaveLength(1);
  return requests[0];
}

describe("authorization", () => {
  it("sends a token for the signed-in admin on every call", async () => {
    const requests = stubFetch(json(200, { tags: [] }));

    await listAdminTags(admin);

    const header = only(requests).headers.get("Authorization") ?? "";
    expect(header).toMatch(/^Bearer /);
    const { payload } = await jwtVerify(header.slice(7), publicKey, {
      issuer: "blog-web",
      audience: "blog-api",
    });
    expect(payload.sub).toBe("github:123");
  });

  it.each([
    [401, "unauthorized" as const],
    [403, "forbidden" as const],
  ])("throws on %i, a key or allowlist mismatch", async (status, code) => {
    stubFetch(apiError(status, code, "no"));

    await expect(
      updateAdminPost(admin, post.id, { ...input, updated_at: LOCK }),
    ).rejects.toEqual(new ApiError(status, code, "no"));
  });
});

describe("listAdminPosts", () => {
  const page: AdminPage = {
    items: [
      {
        id: post.id,
        slug: post.slug,
        title: post.title,
        published_at: null,
        updated_at: LOCK,
        state: "draft",
      },
    ],
    next_cursor: null,
    prev_cursor: null,
  };

  it("sends limit and cursor", async () => {
    const requests = stubFetch(json(200, page));

    await expect(
      listAdminPosts(admin, { limit: 5, cursor: "abc" }),
    ).resolves.toEqual(page);

    const url = new URL(only(requests).url);
    expect(url.pathname).toBe("/api/v1/admin/posts");
    expect(Object.fromEntries(url.searchParams)).toEqual({
      limit: "5",
      cursor: "abc",
    });
  });

  it("returns null when the API rejects the cursor", async () => {
    stubFetch(apiError(400, "invalid_request", "invalid cursor"));

    await expect(listAdminPosts(admin, { cursor: "bad" })).resolves.toBeNull();
  });
});

describe("getAdminPost", () => {
  it("returns the post with its lock token intact", async () => {
    stubFetch(json(200, post));

    const got = await getAdminPost(admin, post.id);

    expect(got?.updated_at).toBe(LOCK);
  });

  it.each([
    [404, "not_found" as const],
    [400, "invalid_request" as const],
  ])("returns null on %i", async (status, code) => {
    stubFetch(apiError(status, code, "no"));

    await expect(getAdminPost(admin, "x")).resolves.toBeNull();
  });
});

describe("createAdminPost", () => {
  it("posts the input and returns the created post", async () => {
    const requests = stubFetch(json(201, post));

    await expect(createAdminPost(admin, input)).resolves.toEqual({
      ok: true,
      value: post,
    });

    const request = only(requests);
    expect(request.method).toBe("POST");
    expect(new URL(request.url).pathname).toBe("/api/v1/admin/posts");
    await expect(request.json()).resolves.toEqual(input);
  });

  it("returns validation problems", async () => {
    const problems = [
      { field: "title", message: "title is required" },
      { field: "tags", message: `tag "!" has no letters or digits` },
    ];
    stubFetch(apiError(400, "invalid_request", "title is required", problems));

    await expect(createAdminPost(admin, input)).resolves.toEqual({
      ok: false,
      error: {
        code: "invalid_request",
        message: "title is required",
        problems,
      },
    });
  });

  it("returns an empty problem list for malformed JSON", async () => {
    stubFetch(apiError(400, "invalid_request", "malformed JSON"));

    const result = await createAdminPost(admin, input);

    expect(result).toEqual({
      ok: false,
      error: {
        code: "invalid_request",
        message: "malformed JSON",
        problems: [],
      },
    });
  });

  it.each([
    [409, "slug_taken" as const],
    [413, "too_large" as const],
  ])("returns %i %s as a value", async (status, code) => {
    stubFetch(apiError(status, code, "no"));

    await expect(createAdminPost(admin, input)).resolves.toEqual({
      ok: false,
      error: { code, message: "no", problems: [] },
    });
  });

  it("throws on a 500", async () => {
    stubFetch(apiError(500, "internal", "boom"));

    await expect(createAdminPost(admin, input)).rejects.toBeInstanceOf(
      ApiError,
    );
  });
});

describe("updateAdminPost", () => {
  it("sends updated_at byte for byte", async () => {
    const requests = stubFetch(json(200, post));

    await updateAdminPost(admin, post.id, { ...input, updated_at: LOCK });

    const request = only(requests);
    expect(request.method).toBe("PUT");
    expect(new URL(request.url).pathname).toBe(
      `/api/v1/admin/posts/${post.id}`,
    );
    expect(await request.text()).toContain(`"updated_at":"${LOCK}"`);
  });

  it.each([
    [409, "conflict" as const],
    [409, "slug_taken" as const],
    [404, "not_found" as const],
    [413, "too_large" as const],
  ])("returns %i %s as a value", async (status, code) => {
    stubFetch(apiError(status, code, "no"));

    const result = await updateAdminPost(admin, post.id, {
      ...input,
      updated_at: LOCK,
    });

    expect(result).toEqual({
      ok: false,
      error: { code, message: "no", problems: [] },
    });
  });
});

describe("previewPost", () => {
  it("returns the rendered HTML", async () => {
    const requests = stubFetch(json(200, { body_html: "<p>Hi</p>" }));

    await expect(previewPost(admin, "Hi")).resolves.toEqual({
      ok: true,
      value: "<p>Hi</p>",
    });

    await expect(only(requests).json()).resolves.toEqual({ body_md: "Hi" });
  });

  it("returns too_large as a value", async () => {
    stubFetch(apiError(413, "too_large", "body too large"));

    const result = await previewPost(admin, "x");

    expect(result.ok).toBe(false);
  });
});

describe("createImageUpload", () => {
  const upload: Upload = {
    url: "http://s3.test/blog/images/abc.png?X-Amz-Signature=1",
    headers: { "Content-Type": "image/png", "Content-Length": "42" },
    public_url: "http://s3.test/blog/images/abc.png",
  };

  it("returns the presigned upload", async () => {
    const requests = stubFetch(json(200, upload));

    await expect(
      createImageUpload(admin, { content_type: "image/png", size: 42 }),
    ).resolves.toEqual({ ok: true, value: upload });

    await expect(only(requests).json()).resolves.toEqual({
      content_type: "image/png",
      size: 42,
    });
  });

  it("returns problems for a rejected type", async () => {
    const problems = [{ field: "content_type", message: "not an image" }];
    stubFetch(apiError(400, "invalid_request", "not an image", problems));

    const result = await createImageUpload(admin, {
      content_type: "text/html",
      size: 42,
    });

    expect(result).toEqual({
      ok: false,
      error: { code: "invalid_request", message: "not an image", problems },
    });
  });
});

describe("listAdminTags", () => {
  it("returns every tag", async () => {
    const tags = [{ slug: "drafts-only", name: "Drafts only" }];
    stubFetch(json(200, { tags }));

    await expect(listAdminTags(admin)).resolves.toEqual(tags);
  });
});
