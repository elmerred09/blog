import type { AdminPostInput, TagColor, UploadInput } from "@/lib/api/types";

// Server actions are public POST endpoints: TypeScript types don't reach the
// wire. These checks keep a malformed call from reaching the API as something
// else; content rules (lengths, slug format, dates) stay in the API, which
// returns them as problems the editor can show.

// id null creates a post. Otherwise updatedAt is the lock token from the post
// the editor loaded, and loadedSlug its slug then, so a rename can expire the
// old page.
export type SavePostInput = {
  id: string | null;
  loadedSlug: string | null;
  updatedAt: string | null;
  post: AdminPostInput;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isString(value: unknown): value is string {
  return typeof value === "string";
}

function isNullableString(value: unknown): value is string | null {
  return value === null || typeof value === "string";
}

function postFields(raw: unknown): AdminPostInput | null {
  if (!isRecord(raw)) {
    return null;
  }
  const { slug, title, summary, body_md, published_at, tags } = raw;
  if (
    !isString(slug) ||
    !isString(title) ||
    !isString(summary) ||
    !isString(body_md) ||
    !(published_at === undefined || isNullableString(published_at)) ||
    !Array.isArray(tags) ||
    !tags.every(isString)
  ) {
    return null;
  }
  return {
    slug,
    title,
    summary,
    body_md,
    published_at: published_at ?? null,
    tags,
  };
}

export function parseSavePostInput(raw: unknown): SavePostInput {
  const post = isRecord(raw) ? postFields(raw.post) : null;
  if (
    !isRecord(raw) ||
    !post ||
    !isNullableString(raw.id) ||
    !isNullableString(raw.loadedSlug) ||
    !isNullableString(raw.updatedAt) ||
    (raw.id === null) !== (raw.updatedAt === null)
  ) {
    throw new Error(
      "invalid save: expected a post and, for an update, its id and updatedAt",
    );
  }
  return {
    id: raw.id,
    loadedSlug: raw.loadedSlug,
    updatedAt: raw.updatedAt,
    post,
  };
}

export function parseUploadInput(raw: unknown): UploadInput {
  if (
    !isRecord(raw) ||
    !isString(raw.content_type) ||
    !Number.isSafeInteger(raw.size)
  ) {
    throw new Error(
      "invalid upload: expected content_type and an integer size",
    );
  }
  return { content_type: raw.content_type, size: raw.size as number };
}

export function parsePreviewInput(raw: unknown): string {
  if (!isString(raw)) {
    throw new Error("invalid preview: expected the markdown body");
  }
  return raw;
}

export type TagColorInput = { slug: string; color: TagColor };

// Whether the color is in the palette is the API's call; it answers with a
// problem on color.
export function parseTagColorInput(raw: unknown): TagColorInput {
  if (!isRecord(raw) || !isString(raw.slug) || !isString(raw.color)) {
    throw new Error("invalid tag color: expected slug and color strings");
  }
  return { slug: raw.slug, color: raw.color as TagColor };
}

// slug is the post's stored slug, whose page the delete expires.
export type DeletePostInput = { id: string; slug: string };

export function parseDeletePostInput(raw: unknown): DeletePostInput {
  if (!isRecord(raw) || !isString(raw.id) || !isString(raw.slug)) {
    throw new Error("invalid delete: expected id and slug strings");
  }
  return { id: raw.id, slug: raw.slug };
}
