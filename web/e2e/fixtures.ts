import type { APIRequestContext } from "@playwright/test";
import { importPKCS8, SignJWT } from "jose";

import type { components } from "../lib/api/schema";
import type { E2EEnv } from "./env";

export type FixturePost = components["schemas"]["AdminPostInput"];
type AdminPost = components["schemas"]["AdminPost"];
type ErrorBody = components["schemas"]["Error"];

// Shared by the published fixtures, so tag pages and tag lists have content.
export const FIXTURE_TAG = "E2E Fixtures";

// A word only the search fixtures contain. The draft and the scheduled post
// have it too, so a search for it shows whether they leak.
export const SEARCH_TERM = "quokka";

// Every published fixture is dated in January 2020. Posts the admin specs
// publish are newer, so they push fixtures down the list but never reorder
// them; specs assert on fixture slugs and their relative order only.
function publishedAt(day: number): string {
  return new Date(Date.UTC(2020, 0, day, 12)).toISOString();
}

export const CODE_POST: FixturePost = {
  slug: "e2e-code-highlighting",
  title: "Highlighting Go code",
  summary: "A fixture with a fenced code block.",
  body_md: [
    "The block below should render with chroma classes.",
    "",
    "```go",
    "package main",
    "",
    'import "fmt"',
    "",
    "func main() {",
    '\tfmt.Println("hello, e2e")',
    "}",
    "```",
  ].join("\n"),
  published_at: publishedAt(14),
  tags: [FIXTURE_TAG, "Go"],
};

export const SEARCH_POST: FixturePost = {
  slug: "e2e-search-quokka",
  title: "Counting quokkas with full-text search",
  summary: "A fixture for the search specs.",
  body_md: `A ${SEARCH_TERM} is a small marsupial. This post exists so a search for ${SEARCH_TERM} has exactly one published match.`,
  published_at: publishedAt(13),
  tags: [FIXTURE_TAG, "Postgres"],
};

// Twelve filler posts, so with the code and search posts there are 14
// published fixtures: more than the home page's 10, so "Older posts" shows.
export const PAGING_POSTS: FixturePost[] = Array.from(
  { length: 12 },
  (_, i) => {
    const n = String(i + 1).padStart(2, "0");
    return {
      slug: `e2e-paging-${n}`,
      title: `Paging fixture ${n}`,
      summary: `Filler post ${n} for keyset paging.`,
      body_md: `Filler post ${n}.`,
      published_at: publishedAt(i + 1),
      tags: [FIXTURE_TAG],
    };
  },
);

export const DRAFT_POST: FixturePost = {
  slug: "e2e-draft",
  title: "Unpublished draft",
  summary: "Must not appear publicly.",
  body_md: `A draft that mentions ${SEARCH_TERM}.`,
  published_at: null,
  tags: [FIXTURE_TAG],
};

export const SCHEDULED_POST: FixturePost = {
  slug: "e2e-scheduled",
  title: "Scheduled for the future",
  summary: "Must not appear publicly until 2099.",
  body_md: `A scheduled post that mentions ${SEARCH_TERM}.`,
  published_at: new Date(Date.UTC(2099, 0, 1, 12)).toISOString(),
  tags: [FIXTURE_TAG],
};

// Newest first, as the home page lists them.
export const PUBLISHED_FIXTURES: FixturePost[] = [
  CODE_POST,
  SEARCH_POST,
  ...PAGING_POSTS.toReversed(),
];

export const ALL_FIXTURES: FixturePost[] = [
  ...PUBLISHED_FIXTURES,
  DRAFT_POST,
  SCHEDULED_POST,
];

// adminToken signs a bearer token like lib/api/token.ts does for the editor.
// That module can't be imported here: its server-only guard throws outside
// Next's server build.
export async function adminToken(env: E2EEnv): Promise<string> {
  const githubId = env.web.ADMIN_GITHUB_IDS.split(",")[0].trim();
  const key = await importPKCS8(
    env.web.ADMIN_JWT_PRIVATE_KEY.replaceAll("\\n", "\n"),
    "EdDSA",
  );
  const iat = Math.floor(Date.now() / 1000);
  return new SignJWT()
    .setProtectedHeader({ alg: "EdDSA", typ: "JWT" })
    .setIssuer("blog-web")
    .setAudience("blog-api")
    .setSubject(`github:${githubId}`)
    .setIssuedAt(iat)
    .setExpirationTime(iat + 120)
    .sign(key);
}

function expectedState(post: FixturePost): AdminPost["state"] {
  if (!post.published_at) {
    return "draft";
  }
  return Date.parse(post.published_at) > Date.now() ? "scheduled" : "published";
}

// createFixture creates the post through the admin API. It returns false when
// the slug is already taken, which means an earlier run created it against
// a database that wasn't reset (a rerun without make e2e).
export async function createFixture(
  request: APIRequestContext,
  env: E2EEnv,
  post: FixturePost,
): Promise<boolean> {
  const res = await request.post(`${env.web.API_URL}/api/v1/admin/posts`, {
    headers: { authorization: `Bearer ${await adminToken(env)}` },
    data: post,
  });
  if (res.status() === 201) {
    const { state } = (await res.json()) as AdminPost;
    const want = expectedState(post);
    if (state !== want) {
      throw new Error(`fixture ${post.slug}: state ${state}, want ${want}`);
    }
    return true;
  }
  if (res.status() === 409) {
    const body = (await res.json()) as ErrorBody;
    if (body.error.code === "slug_taken") {
      return false;
    }
  }
  throw new Error(
    `creating fixture ${post.slug}: ${res.status()} ${await res.text()}`,
  );
}
