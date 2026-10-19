// @vitest-environment node
import { updateTag } from "next/cache";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { requireAdmin, type Admin } from "@/lib/admin/session";
import * as api from "@/lib/api/admin";
import type { AdminPost, AdminPostInput } from "@/lib/api/types";

import {
  createImageUpload,
  previewPost,
  savePost,
  type SavePostInput,
} from "./actions";

vi.mock("server-only", () => ({}));
vi.mock("next/cache", () => ({ updateTag: vi.fn() }));
vi.mock("@/lib/admin/session", () => ({ requireAdmin: vi.fn() }));
vi.mock("@/lib/api/admin", () => ({
  createAdminPost: vi.fn(),
  updateAdminPost: vi.fn(),
  previewPost: vi.fn(),
  createImageUpload: vi.fn(),
}));

const admin: Admin = { githubId: "123", name: "Elmer" };
const LOCK = "2026-10-18T12:00:12.345678Z";
const ID = "0f8a5c1e-4b2d-4c3a-9e1f-2a3b4c5d6e7f";

const fields: AdminPostInput = {
  slug: "keyset-pagination",
  title: "Keyset pagination",
  summary: "Why OFFSET gets slow.",
  body_md: "Hello.",
  published_at: "2026-10-20T09:00:00Z",
  tags: ["Postgres"],
};

function saved(slug: string): AdminPost {
  return {
    id: ID,
    ...fields,
    slug,
    published_at: fields.published_at ?? null,
    created_at: "2026-10-18T11:00:00Z",
    updated_at: "2026-10-18T12:05:00.000001Z",
    state: "scheduled",
    tags: [{ slug: "postgres", name: "Postgres" }],
  };
}

const update: SavePostInput = {
  id: ID,
  loadedSlug: "keyset-pagination",
  updatedAt: LOCK,
  post: fields,
};

// requireAdmin redirects or 404s by throwing, like next/navigation does.
function signedOut() {
  vi.mocked(requireAdmin).mockRejectedValue(new Error("NEXT_REDIRECT"));
}

function tags(): string[] {
  return vi.mocked(updateTag).mock.calls.map(([tag]) => tag);
}

beforeEach(() => {
  vi.mocked(requireAdmin).mockResolvedValue(admin);
});

describe("savePost", () => {
  it("creates a post and expires the list and its page", async () => {
    vi.mocked(api.createAdminPost).mockResolvedValue({
      ok: true,
      value: saved(fields.slug),
    });

    const result = await savePost({
      id: null,
      loadedSlug: null,
      updatedAt: null,
      post: fields,
    });

    expect(result.ok).toBe(true);
    expect(api.createAdminPost).toHaveBeenCalledExactlyOnceWith(admin, fields);
    expect(tags().sort()).toEqual(["post:keyset-pagination", "posts"]);
  });

  it("updates with the loaded lock token unchanged", async () => {
    vi.mocked(api.updateAdminPost).mockResolvedValue({
      ok: true,
      value: saved(fields.slug),
    });

    await savePost(update);

    expect(api.updateAdminPost).toHaveBeenCalledExactlyOnceWith(admin, ID, {
      ...fields,
      updated_at: LOCK,
    });
    expect(tags().sort()).toEqual(["post:keyset-pagination", "posts"]);
  });

  it("expires the old slug's page on a rename", async () => {
    vi.mocked(api.updateAdminPost).mockResolvedValue({
      ok: true,
      value: saved("keyset-pagination-in-postgres"),
    });

    await savePost({
      ...update,
      post: { ...fields, slug: "keyset-pagination-in-postgres" },
    });

    expect(tags().sort()).toEqual([
      "post:keyset-pagination",
      "post:keyset-pagination-in-postgres",
      "posts",
    ]);
  });

  it("returns a failure without expiring anything", async () => {
    const failure = {
      ok: false as const,
      error: { code: "conflict" as const, message: "stale", problems: [] },
    };
    vi.mocked(api.updateAdminPost).mockResolvedValue(failure);

    await expect(savePost(update)).resolves.toEqual(failure);
    expect(updateTag).not.toHaveBeenCalled();
  });

  it("requires an admin before calling the API", async () => {
    signedOut();

    await expect(savePost(update)).rejects.toThrow("NEXT_REDIRECT");
    expect(api.updateAdminPost).not.toHaveBeenCalled();
    expect(updateTag).not.toHaveBeenCalled();
  });

  it("forwards only the post's fields", async () => {
    vi.mocked(api.createAdminPost).mockResolvedValue({
      ok: true,
      value: saved(fields.slug),
    });

    await savePost({
      id: null,
      loadedSlug: null,
      updatedAt: null,
      post: { ...fields, state: "published" } as AdminPostInput,
    });

    expect(api.createAdminPost).toHaveBeenCalledExactlyOnceWith(admin, fields);
  });

  it.each<[string, unknown]>([
    ["an update without a lock token", { ...update, updatedAt: null }],
    ["a lock token on a create", { ...update, id: null }],
    ["a non-string title", { ...update, post: { ...fields, title: 1 } }],
    ["tags that aren't strings", { ...update, post: { ...fields, tags: [1] } }],
    ["a missing post", { ...update, post: undefined }],
    ["no input", undefined],
  ])("rejects %s", async (_, input) => {
    await expect(savePost(input as SavePostInput)).rejects.toThrow(
      /invalid save/,
    );
    expect(api.updateAdminPost).not.toHaveBeenCalled();
    expect(api.createAdminPost).not.toHaveBeenCalled();
  });

  it("accepts an omitted published_at as a draft", async () => {
    vi.mocked(api.createAdminPost).mockResolvedValue({
      ok: true,
      value: saved(fields.slug),
    });
    const draft = { ...fields };
    delete draft.published_at;

    await savePost({
      id: null,
      loadedSlug: null,
      updatedAt: null,
      post: draft,
    });

    expect(api.createAdminPost).toHaveBeenCalledExactlyOnceWith(admin, {
      ...draft,
      published_at: null,
    });
  });
});

describe("previewPost", () => {
  it("renders through the API", async () => {
    vi.mocked(api.previewPost).mockResolvedValue({ ok: true, value: "<p/>" });

    await expect(previewPost("Hi")).resolves.toEqual({
      ok: true,
      value: "<p/>",
    });
    expect(api.previewPost).toHaveBeenCalledExactlyOnceWith(admin, "Hi");
  });

  it("requires an admin", async () => {
    signedOut();

    await expect(previewPost("Hi")).rejects.toThrow("NEXT_REDIRECT");
    expect(api.previewPost).not.toHaveBeenCalled();
  });

  it("rejects a non-string body", async () => {
    await expect(previewPost(1 as unknown as string)).rejects.toThrow(
      /invalid preview/,
    );
  });
});

describe("createImageUpload", () => {
  const upload = { content_type: "image/png", size: 42 };

  it("asks the API for a presigned upload", async () => {
    const value = {
      url: "http://s3.test/put",
      headers: { "Content-Type": "image/png", "Content-Length": "42" },
      public_url: "http://s3.test/get",
    };
    vi.mocked(api.createImageUpload).mockResolvedValue({ ok: true, value });

    await expect(createImageUpload(upload)).resolves.toEqual({
      ok: true,
      value,
    });
    expect(api.createImageUpload).toHaveBeenCalledExactlyOnceWith(
      admin,
      upload,
    );
  });

  it("requires an admin", async () => {
    signedOut();

    await expect(createImageUpload(upload)).rejects.toThrow("NEXT_REDIRECT");
    expect(api.createImageUpload).not.toHaveBeenCalled();
  });

  it.each<[string, unknown]>([
    ["a fractional size", { ...upload, size: 1.5 }],
    ["a string size", { ...upload, size: "42" }],
    ["a missing type", { size: 42 }],
  ])("rejects %s", async (_, input) => {
    await expect(createImageUpload(input as typeof upload)).rejects.toThrow(
      /invalid upload/,
    );
  });
});
