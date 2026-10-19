import "server-only";

import type { Admin } from "@/lib/admin/session";

import { apiClient, unwrap, type Result } from "./client";
import { apiErrorFrom } from "./errors";
import { signApiToken } from "./token";
import type {
  AdminPage,
  AdminPost,
  AdminPostInput,
  AdminPostUpdate,
  ErrorCode,
  Problem,
  Tag,
  Upload,
  UploadInput,
} from "./types";

// The admin client is never cached: each call carries a fresh token for the
// signed-in admin, and the editor must see its own writes. Callers get the
// Admin from requireAdmin.
//
// updated_at is the optimistic-lock token. It passes through as the string
// the API sent; parsing it into a Date would drop the microseconds and turn
// every save into a conflict.

export type ListAdminPostsParams = {
  limit?: number;
  cursor?: string;
};

const failureCodes = [
  "invalid_request",
  "not_found",
  "conflict",
  "slug_taken",
  "too_large",
] as const satisfies readonly ErrorCode[];

export type FailureCode = (typeof failureCodes)[number];

// problems is empty unless input failed validation, such as malformed JSON
// or a bad id.
export type Failure = {
  code: FailureCode;
  message: string;
  problems: Problem[];
};

// Expected failures are values, so a server action can return them to the
// editor. Anything else, such as a 401 from a key mismatch, throws ApiError.
export type Outcome<T> = { ok: true; value: T } | { ok: false; error: Failure };

function isFailureCode(code: string): code is FailureCode {
  return (failureCodes as readonly string[]).includes(code);
}

function outcome<T>({ data, error, response }: Result<T>): Outcome<T> {
  if (response.ok && data !== undefined) {
    return { ok: true, value: data };
  }
  const err = apiErrorFrom(response.status, error);
  if (!isFailureCode(err.code)) {
    throw err;
  }
  const problems = (error as { error: { problems?: Problem[] } }).error
    .problems;
  return {
    ok: false,
    error: { code: err.code, message: err.message, problems: problems ?? [] },
  };
}

async function authorization(admin: Admin) {
  return { Authorization: `Bearer ${await signApiToken(admin.githubId)}` };
}

// Returns null on 400: a cursor or limit the API rejects.
export async function listAdminPosts(
  admin: Admin,
  params: ListAdminPostsParams = {},
): Promise<AdminPage | null> {
  const result = await apiClient().GET("/api/v1/admin/posts", {
    params: { query: params },
    headers: await authorization(admin),
  });
  if (result.response.status === 400) {
    return null;
  }
  return unwrap(result);
}

// Returns null on 404, and on 400 for an id that isn't a UUID.
export async function getAdminPost(
  admin: Admin,
  id: string,
): Promise<AdminPost | null> {
  const result = await apiClient().GET("/api/v1/admin/posts/{id}", {
    params: { path: { id } },
    headers: await authorization(admin),
  });
  if (result.response.status === 404 || result.response.status === 400) {
    return null;
  }
  return unwrap(result);
}

export async function createAdminPost(
  admin: Admin,
  input: AdminPostInput,
): Promise<Outcome<AdminPost>> {
  return outcome(
    await apiClient().POST("/api/v1/admin/posts", {
      body: input,
      headers: await authorization(admin),
    }),
  );
}

export async function updateAdminPost(
  admin: Admin,
  id: string,
  input: AdminPostUpdate,
): Promise<Outcome<AdminPost>> {
  return outcome(
    await apiClient().PUT("/api/v1/admin/posts/{id}", {
      params: { path: { id } },
      body: input,
      headers: await authorization(admin),
    }),
  );
}

export async function previewPost(
  admin: Admin,
  bodyMd: string,
): Promise<Outcome<string>> {
  const result = outcome(
    await apiClient().POST("/api/v1/admin/preview", {
      body: { body_md: bodyMd },
      headers: await authorization(admin),
    }),
  );
  return result.ok ? { ok: true, value: result.value.body_html } : result;
}

export async function createImageUpload(
  admin: Admin,
  input: UploadInput,
): Promise<Outcome<Upload>> {
  return outcome(
    await apiClient().POST("/api/v1/admin/uploads", {
      body: input,
      headers: await authorization(admin),
    }),
  );
}

// Every tag, including ones only drafts use.
export async function listAdminTags(admin: Admin): Promise<Tag[]> {
  const result = await apiClient().GET("/api/v1/admin/tags", {
    headers: await authorization(admin),
  });
  return unwrap(result).tags;
}
