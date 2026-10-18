-- name: UpsertPost :one
-- index: posts_slug_key (unique on slug).
INSERT INTO posts (slug, title, summary, body_md, body_html, published_at)
VALUES (
  sqlc.arg(slug),
  sqlc.arg(title),
  sqlc.arg(summary),
  sqlc.arg(body_md),
  sqlc.arg(body_html),
  sqlc.arg(published_at)
)
ON CONFLICT (slug) DO UPDATE
SET
  title = EXCLUDED.title,
  summary = EXCLUDED.summary,
  body_md = EXCLUDED.body_md,
  body_html = EXCLUDED.body_html,
  published_at = EXCLUDED.published_at,
  deleted_at = NULL,
  updated_at = CURRENT_TIMESTAMP
WHERE (
    (posts.title, posts.summary, posts.body_md, posts.body_html, posts.published_at)
    IS DISTINCT FROM
    (EXCLUDED.title, EXCLUDED.summary, EXCLUDED.body_md, EXCLUDED.body_html, EXCLUDED.published_at)
  )
  OR posts.deleted_at IS NOT NULL
RETURNING id, (old.id IS NULL)::boolean AS created;

-- name: GetPostBySlug :one
-- index: posts_slug_key (unique on slug).
SELECT id, slug, title, summary, body_html, published_at, deleted_at, created_at, updated_at
FROM posts
WHERE slug = sqlc.arg(slug)
    AND published_at <= CURRENT_TIMESTAMP
    AND deleted_at IS NULL
LIMIT 1;

-- name: GetPostIDBySlug :one
-- index: posts_slug_key (unique on slug).
SELECT id
FROM posts
WHERE slug = sqlc.arg(slug)
LIMIT 1;

-- name: ListPosts :many
-- index: posts_pagination_idx (published_at DESC, id DESC) WHERE published_at IS NOT NULL AND deleted_at IS NULL.
SELECT id, slug, title, summary, published_at
FROM posts
WHERE published_at <= CURRENT_TIMESTAMP
  AND deleted_at IS NULL
  AND (published_at, id) < (
    COALESCE(sqlc.narg(after_published_at)::timestamptz, 'infinity'),
    COALESCE(sqlc.narg(after_id)::uuid, 'ffffffff-ffff-ffff-ffff-ffffffffffff')
  )
ORDER BY published_at DESC, id DESC
LIMIT sqlc.arg(page_size);

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

-- name: ListAdminPosts :many
-- index: posts_pkey (id), scanned backward from after_id.
SELECT id, slug, title, published_at, updated_at
FROM posts
WHERE deleted_at IS NULL
  AND id < COALESCE(sqlc.narg(after_id)::uuid, 'ffffffff-ffff-ffff-ffff-ffffffffffff')
ORDER BY id DESC
LIMIT sqlc.arg(page_size);

-- name: SoftDeletePost :one
-- index: posts_pkey (id).
UPDATE posts
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
  AND deleted_at IS NULL
RETURNING id;
