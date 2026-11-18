-- name: GetPostBySlug :one
-- index: posts_slug_key (unique on slug).
SELECT id, slug, title, summary, body_html, word_count, published_at, deleted_at, created_at, updated_at
FROM posts
WHERE slug = sqlc.arg(slug)
    AND published_at <= CURRENT_TIMESTAMP
    AND deleted_at IS NULL
LIMIT 1;

-- name: ListPosts :many
-- index: posts_pagination_idx (published_at DESC, id DESC) WHERE published_at IS NOT NULL AND deleted_at IS NULL.
SELECT id, slug, title, summary, published_at, word_count
FROM posts
WHERE published_at <= CURRENT_TIMESTAMP
  AND deleted_at IS NULL
  AND (published_at, id) < (
    COALESCE(sqlc.narg(after_published_at)::timestamptz, 'infinity'),
    COALESCE(sqlc.narg(after_id)::uuid, 'ffffffff-ffff-ffff-ffff-ffffffffffff')
  )
ORDER BY published_at DESC, id DESC
LIMIT sqlc.arg(page_size);

-- name: ListPostsBefore :many
-- index: posts_pagination_idx (published_at DESC, id DESC) WHERE published_at IS NOT NULL AND deleted_at IS NULL,
-- scanned backward: ASC is the reverse of the index order. The caller reverses the rows.
-- A backward page always has a cursor, so the comparison needs no NULL sentinel.
SELECT id, slug, title, summary, published_at, word_count
FROM posts
WHERE published_at <= CURRENT_TIMESTAMP
  AND deleted_at IS NULL
  AND (published_at, id) > (
    sqlc.arg(before_published_at)::timestamptz,
    sqlc.arg(before_id)::uuid
  )
ORDER BY published_at ASC, id ASC
LIMIT sqlc.arg(page_size);

-- name: GetAdjacentPosts :one
-- index: posts_slug_key finds the current post. Each subquery reads one entry of
-- posts_pagination_idx (published_at DESC, id DESC) WHERE published_at IS NOT NULL
-- AND deleted_at IS NULL, starting at the current post's (published_at, id):
-- older reads forward, newer reads backward (ASC is the reverse of the index).
-- posts_pkey then fetches that neighbor's row. The joins are on posts rather
-- than LATERAL subqueries so sqlc types the neighbor columns as nullable.
-- No row means the current post isn't published; a NULL side means there is
-- no post in that direction.
SELECT newer.slug AS newer_slug, newer.title AS newer_title,
       older.slug AS older_slug, older.title AS older_title
FROM posts cur
LEFT JOIN posts newer ON newer.id = (
  SELECT p.id
  FROM posts p
  WHERE p.published_at <= CURRENT_TIMESTAMP
    AND p.deleted_at IS NULL
    AND (p.published_at, p.id) > (cur.published_at, cur.id)
  ORDER BY p.published_at ASC, p.id ASC
  LIMIT 1
)
LEFT JOIN posts older ON older.id = (
  SELECT p.id
  FROM posts p
  WHERE p.published_at <= CURRENT_TIMESTAMP
    AND p.deleted_at IS NULL
    AND (p.published_at, p.id) < (cur.published_at, cur.id)
  ORDER BY p.published_at DESC, p.id DESC
  LIMIT 1
)
WHERE cur.slug = sqlc.arg(slug)
  AND cur.published_at <= CURRENT_TIMESTAMP
  AND cur.deleted_at IS NULL;

-- name: CreatePost :one
-- index: posts_slug_key (unique on slug).
INSERT INTO posts (slug, title, summary, body_md, body_html, published_at)
VALUES (
  sqlc.arg(slug),
  sqlc.arg(title),
  sqlc.arg(summary),
  sqlc.arg(body_md),
  sqlc.arg(body_html),
  sqlc.narg(published_at)
)
RETURNING id, updated_at;

-- name: UpdatePost :one
-- index: posts_pkey (id) for both the UPDATE and the EXISTS probe; posts_slug_key
-- when the slug changes (23505 if it's taken).
WITH updated AS (
  UPDATE posts
  SET
    slug = sqlc.arg(slug),
    title = sqlc.arg(title),
    summary = sqlc.arg(summary),
    body_md = sqlc.arg(body_md),
    body_html = sqlc.arg(body_html),
    published_at = sqlc.narg(published_at),
    updated_at = CURRENT_TIMESTAMP
  WHERE posts.id = sqlc.arg(id)
    AND posts.deleted_at IS NULL
    AND posts.updated_at = sqlc.arg(expected_updated_at)
  RETURNING posts.updated_at
)
SELECT
  EXISTS (SELECT 1 FROM posts p WHERE p.id = sqlc.arg(id) AND p.deleted_at IS NULL) AS found,
  u.updated_at
FROM (SELECT 1) AS one
LEFT JOIN updated u ON true;

-- name: GetPostByID :one
-- index: posts_pkey (id).
SELECT id, slug, title, summary, body_md, published_at, created_at, updated_at
FROM posts
WHERE id = sqlc.arg(id)
  AND deleted_at IS NULL;

-- name: ListPostBodies :many
-- index: posts_pkey (id), scanned forward from after_id. The first batch
-- passes the zero UUID, which sorts before every id.
SELECT id, body_md, body_html
FROM posts
WHERE deleted_at IS NULL
  AND id > sqlc.arg(after_id)
ORDER BY id ASC
LIMIT sqlc.arg(batch_size);

-- name: SetPostHTML :execrows
-- index: posts_pkey (id).
-- Matching body_md skips a post an editor saved since ListPostBodies read it;
-- that save stored HTML from the current renderer. updated_at stays put: it is
-- the editor's optimistic-lock token, and the post's content did not change.
UPDATE posts
SET body_html = sqlc.arg(body_html)
WHERE id = sqlc.arg(id)
  AND body_md = sqlc.arg(body_md)
  AND deleted_at IS NULL;

-- name: ListAdminPosts :many
-- index: posts_pkey (id), scanned backward from after_id.
SELECT id, slug, title, published_at, updated_at
FROM posts
WHERE deleted_at IS NULL
  AND id < COALESCE(sqlc.narg(after_id)::uuid, 'ffffffff-ffff-ffff-ffff-ffffffffffff')
ORDER BY id DESC
LIMIT sqlc.arg(page_size);

-- name: ListAdminPostsBefore :many
-- index: posts_pkey (id), scanned forward from before_id. The caller reverses the rows.
SELECT id, slug, title, published_at, updated_at
FROM posts
WHERE deleted_at IS NULL
  AND id > sqlc.arg(before_id)
ORDER BY id ASC
LIMIT sqlc.arg(page_size);

-- name: ListSitemapPosts :many
-- A seq scan is acceptable. Nearly every row is published, and slug and
-- updated_at are not in posts_pagination_idx, so walking that index and
-- fetching the heap costs more than reading the table (about 9ms and an
-- in-memory quicksort on the 10k seed; forcing the index was slower).
-- A covering index would be an index-only scan a few milliseconds faster,
-- which is not worth a migration for a list cached for hours.
-- last_modified is the later of updated_at and published_at. A post can be
-- edited while it is still scheduled, so updated_at can be earlier than the
-- moment a reader can see it, which is published_at. An edit after that
-- moves updated_at past published_at. GREATEST is that later moment.
-- Postgres ignores nulls in GREATEST, and updated_at is NOT NULL, so the
-- value is never null. COALESCE is there so sqlc's type is time.Time.
-- 50000 is one sitemap file's URL limit. Nothing pages past it.
SELECT slug,
       COALESCE(GREATEST(updated_at, published_at), updated_at) AS last_modified
FROM posts
WHERE published_at <= CURRENT_TIMESTAMP
  AND deleted_at IS NULL
ORDER BY published_at DESC, id DESC
LIMIT 50000;

-- name: SoftDeletePost :one
-- index: posts_pkey (id).
UPDATE posts
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
  AND deleted_at IS NULL
RETURNING id;
