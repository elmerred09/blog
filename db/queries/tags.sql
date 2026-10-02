-- name: UpsertTag :execrows
-- index: tags_slug_key (unique on slug) is the ON CONFLICT arbiter.
INSERT INTO tags (slug, name)
VALUES (sqlc.arg(slug), sqlc.arg(name))
ON CONFLICT (slug) DO NOTHING;

-- name: GetTagsBySlugs :many
-- index: tags_slug_key (unique on slug), one probe per slug in the array.
SELECT id, slug, name
FROM tags
WHERE slug = ANY(sqlc.arg(slugs)::text[])
ORDER BY slug ASC;

-- name: ListTagsForPost :many
-- index: post_tags_pkey (post_id, tag_id) finds the post's rows by its leading column; tags_pkey for the join.
-- ORDER BY t.slug sorts in memory, fine for a handful of tags per post.
SELECT t.id, t.slug, t.name
FROM tags t
JOIN post_tags pt ON t.id = pt.tag_id
WHERE pt.post_id = sqlc.arg(post_id)
ORDER BY t.slug ASC;

-- name: DeleteTagsForPost :exec
-- index: post_tags_pkey (post_id, tag_id), leading column.
DELETE FROM post_tags
WHERE post_id = sqlc.arg(post_id);

-- name: SetTagsForPost :exec
-- index: post_tags_pkey (post_id, tag_id).
INSERT INTO post_tags (post_id, tag_id)
SELECT sqlc.arg(post_id), UNNEST(sqlc.arg(tag_ids)::int[])
ON CONFLICT (post_id, tag_id) DO NOTHING;

-- name: ListPostsByTag :many
-- index: posts_pagination_idx (published_at DESC, id DESC) WHERE published_at IS NOT NULL AND deleted_at IS NULL.
SELECT p.id, p.slug, p.title, p.published_at
FROM posts p
JOIN post_tags pt ON p.id = pt.post_id
WHERE pt.tag_id = sqlc.arg(tag_id)
  AND p.published_at <= now()
  AND p.deleted_at IS NULL
  AND (p.published_at, p.id) < (
    coalesce(sqlc.narg(after_published_at)::timestamptz, 'infinity'),
    coalesce(sqlc.narg(after_id)::uuid, 'ffffffff-ffff-ffff-ffff-ffffffffffff')
  )
ORDER BY p.published_at DESC, p.id DESC
LIMIT sqlc.arg(page_size);
