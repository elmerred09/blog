TRUNCATE posts, tags RESTART IDENTITY CASCADE;

INSERT INTO posts (slug, title, summary, body_md, body_html, published_at, deleted_at)
SELECT
    'post-' || g AS slug,
    'Post ' || g AS title,
    'Summary ' || g AS summary,
    REPEAT('some text ', 200) AS body_md,
    REPEAT('<p>some text </p>', 200) AS body_html,
    CASE
        WHEN g % 20 = 3 THEN NULL                                        -- 5% drafts
        WHEN g % 50 = 7 THEN NOW() + INTERVAL '1 minute' * g             -- 2% scheduled
        ELSE DATE_TRUNC('day', NOW() - INTERVAL '1 hour' * g)            -- ~24 ties per day
    END AS published_at,
    CASE WHEN g % 33 = 5 THEN NOW() END AS deleted_at                    -- 3% soft-deleted
FROM generate_series(1, 10000) AS g;

INSERT INTO tags (slug, name)
VALUES ('common', 'Common'), ('middle', 'Middle'), ('rare', 'Rare'), ('rare-old', 'Rare Old');

-- common:   90%, everywhere
-- middle:   20%, everywhere
-- rare:      1%, spread evenly through time
-- rare-old:  1%, only the 100 oldest posts
INSERT INTO post_tags (post_id, tag_id)
SELECT p.id, t.id
FROM (SELECT id, SPLIT_PART(slug, '-', 2)::int AS g FROM posts) p
JOIN tags t ON
       (t.slug = 'common'   AND p.g % 10 <> 0)
    OR (t.slug = 'middle'   AND p.g % 5 = 1)
    OR (t.slug = 'rare'     AND p.g % 100 = 1)
    OR (t.slug = 'rare-old' AND p.g > 9900);

VACUUM ANALYZE posts, tags, post_tags;
