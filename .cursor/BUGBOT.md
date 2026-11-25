# Review rules

A technical blog: Go API (gin, pgx, sqlc, goose) on Postgres, and a Next.js 16 App Router frontend in `web/`. One maintainer; most pull requests are opened by coding agents.

A merge to `main` deploys to production. The deploy runs `migrate up`, then starts the new containers, so for a short time the previously deployed API and web run against the new schema.

Report bugs, security problems, and breaks of the rules below. Don't comment on formatting or anything gofmt, Prettier, golangci-lint, or ESLint already enforce, and don't suggest refactors the change doesn't need.

## Generated files

- `internal/db/` (sqlc), `web/lib/api/schema.d.ts` (from `api/openapi.yaml`), and `web/app/chroma.css` (from `internal/render`) are generated. Flag a change to them that has no matching change to `db/queries` or `db/migrations`, `api/openapi.yaml`, or the themes in `internal/render/render.go`.

## Migrations and deploy safety

- Migrations in `db/migrations` are append-only. Flag any edit to an existing migration.
- Every migration needs `-- +goose Up` and `-- +goose Down`, and the Down must reverse the Up.
- Flag a migration the currently deployed code can't run against: dropping or renaming a column or table that existing queries still use, or adding a `NOT NULL` column without a default that existing inserts don't set.
- Point out migrations that rewrite a table or hold a long lock on `posts` or `tags` (stored generated columns, type changes, index builds), so the pull request says so.

## SQL queries (`db/queries`)

- Every query that filters, joins, or sorts has a comment naming the index it uses, or saying why a scan is fine. Flag a new or changed query without one, or whose comment no longer matches.
- No `SELECT *`; no `OFFSET`. Lists page by keyset on `(published_at, id)`.
- An optional cursor uses `coalesce(sqlc.narg(...), sentinel)`. Flag `sqlc.narg(x) IS NULL OR ...` around a condition an index should serve.
- Post state comes from timestamps, there is no status column. A public query must include `published_at <= now()` (or `CURRENT_TIMESTAMP`) and `deleted_at IS NULL`; missing either leaks drafts, scheduled, or deleted posts.
- Search uses `websearch_to_tsquery` and `ts_rank_cd`, and runs `ts_headline` only on the top-N ranked rows, never the whole match set.
- Tags are looked up and deduplicated by `slug`; `name` is display text only.
- Never build SQL by string concatenation.

## Go

- Errors are wrapped with `%w` and compared with `errors.Is` or `errors.As`, never `==` or string matching.
- Functions that do I/O take `context.Context` first. Handlers pass `c.Request.Context()` to services, never the `*gin.Context`. No `context.Background()` outside `main` and tests.
- Gin types stay inside `internal/httpapi`. `httpapi` gets services or interfaces, never a raw `*db.Queries`.
- `internal/post` is read-only; post writes belong in `internal/admin`.
- Rejected input is returned as `*validation.Error` (the handler maps it to a 400); flag new ad hoc validation error types.
- No global mutable state or `init()` side effects.
- Never put query arguments, tokens, or cookies on a span or in a log line.
- Integration tests use a real Postgres through testcontainers, not mocks.

## API contract

- `api/openapi.yaml` is the contract and changes with the handlers. Flag a response field, route, status, or error code that changed in one but not the other.
- Removing, renaming, or newly requiring a response field breaks the web client: check `web/lib/api` and its test fixtures were updated, and that `make web-types` output is included.

## Next.js (`web/`)

- Caching lives only in `"use cache"` functions in `web/lib/api`, each with an explicit `cacheLife(...)` and `cacheTag(...)`. Flag `fetch` caching options (`cache: 'force-cache'`, `next: { revalidate, tags }`) and route segment exports (`dynamic`, `revalidate`, `fetchCache`).
- An error thrown inside a `"use cache"` function reaches the caller as a plain `Error`, so `instanceof ApiError` on it never matches. Expected outcomes (not found, invalid input) must be returned, not thrown.
- Cached functions must not read `cookies()`, `headers()`, or `searchParams`.
- Only `web/lib/api` calls the Go API. No ad hoc `fetch` to it from components, and no `any`.
- Server components by default; `"use client"` only for state, effects, or browser APIs.
- Times shown in the reader's local zone render only in client-only components, or hydration mismatches.

## Admin and auth

- Every page under `web/app/admin` and every admin server action calls `requireAdmin()`. A layout check or `proxy.ts` is not enough. Flag a new admin page or action without it.
- The session is read only in `web/lib/admin/session.ts`. Never trust a GitHub id sent by the client.
- Server actions check their argument's shape at runtime (`web/lib/admin/input.ts`); TypeScript types don't reach the wire.
- The admin API is called only through `web/lib/api/admin.ts`, which is uncached and not exported from the `web/lib/api` barrel.
- `updated_at` is an opaque optimistic-lock token. Flag `new Date(updated_at)`, `toISOString()`, or any reformatting of it: it drops microseconds and turns every save into a 409.
- After an admin write, the action expires the `posts` tag and `post:<slug>`, plus `post:<old slug>` on a rename.
- Admin tokens are EdDSA and live two minutes. Flag a longer lifetime, a different algorithm, or skipped issuer, audience, or expiry checks in the API.
- The editor Back link confirms before discarding unsaved changes. Flag a Back link that navigates straight away, or a discard dialog nested inside the post form.
- Tiptap is destroyed when its route is hidden and rebuilt from its `content` option. Flag an editor whose content is the markdown from the first render: saved body edits then disappear until a full reload.

## Rendering and security

- Post HTML is rendered by `internal/render` (goldmark, raw HTML escaped) when a post is saved. `web/components/post-body.tsx` is the only place that may use `dangerouslySetInnerHTML`; flag it anywhere else, and flag rendering paths that skip `internal/render`.
- Every Tiptap editor extension needs a markdown form goldmark renders (`web/lib/admin/editor-extensions.ts`).
- No secrets, keys, or `.env` files in the diff. A new environment variable needs an entry in the matching `deploy/prod/*.env.example`.

## End-to-end specs (`web/e2e`)

- Nothing requests a public page before the `fixtures` project runs.
- Admin specs import `test` from `e2e/admin.ts`, create their own posts under `uniqueSlug`, and read only those.
- Public specs assert on fixture posts and their relative order, never on counts or positions.
- Assert on the final URL and visible text, never on status codes.
- Only the search fixtures, the draft, and the scheduled post contain "quokka".
