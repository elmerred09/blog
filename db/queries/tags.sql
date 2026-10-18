-- name: UpsertTag :execrows
-- index: tags_slug_key (unique on slug).
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

-- name: ListTagsForPosts :many
-- index: post_tags_pkey (post_id, tag_id).
SELECT pt.post_id, t.slug, t.name
FROM tags t
JOIN post_tags pt ON t.id = pt.tag_id
WHERE pt.post_id = ANY(sqlc.arg(post_ids)::uuid[])
ORDER BY t.slug ASC;

-- name: DeleteTagsForPostExcept :execrows
-- index: post_tags_pkey (post_id, tag_id), leading column.
-- A NULL array would make NOT (tag_id = ANY(NULL)) NULL and delete nothing,
-- so treat it as empty: no tags means remove every link.
DELETE FROM post_tags
WHERE post_id = sqlc.arg(post_id)
  AND NOT (tag_id = ANY(COALESCE(sqlc.arg(tag_ids)::int[], '{}')));

-- name: SetTagsForPost :execrows
-- index: post_tags_pkey (post_id, tag_id).
INSERT INTO post_tags (post_id, tag_id)
SELECT sqlc.arg(post_id), UNNEST(sqlc.arg(tag_ids)::int[])
ON CONFLICT (post_id, tag_id) DO NOTHING;

-- name: ListTags :many
-- index: posts_pagination_idx (index-only).
-- With published_only, a tag with no published post is dropped after the
-- count, so the public list can't reveal a draft's tags. Without it, every
-- tag is listed, for the editor. Both read all of tags, a small table.
SELECT t.id, t.slug, t.name, count(p.id) AS published_posts
FROM tags t
LEFT JOIN post_tags pt ON pt.tag_id = t.id
LEFT JOIN posts p
  ON p.id = pt.post_id
  AND p.published_at <= CURRENT_TIMESTAMP
  AND p.deleted_at IS NULL
GROUP BY t.id
HAVING NOT sqlc.arg(published_only)::boolean OR count(p.id) > 0
ORDER BY t.slug ASC;

-- name: ListPostsByTag :many
-- index: posts_pagination_idx (published_at DESC, id DESC) WHERE published_at IS NOT NULL AND deleted_at IS NULL.
SELECT p.id, p.slug, p.title, p.published_at
FROM posts p
JOIN post_tags pt ON p.id = pt.post_id
WHERE pt.tag_id = sqlc.arg(tag_id)
  AND p.published_at <= CURRENT_TIMESTAMP
  AND p.deleted_at IS NULL
  AND (p.published_at, p.id) < (
    COALESCE(sqlc.narg(after_published_at)::timestamptz, 'infinity'),
    COALESCE(sqlc.narg(after_id)::uuid, 'ffffffff-ffff-ffff-ffff-ffffffffffff')
  )
ORDER BY p.published_at DESC, p.id DESC
LIMIT sqlc.arg(page_size);
