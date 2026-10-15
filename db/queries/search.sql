-- name: SearchPosts :many
-- index: posts_search_idx (GIN on search_vector) for selective terms. A term
-- in most posts is a seq scan, which the planner prefers since every match has
-- to be ranked anyway; the sort is a top-N heapsort over the matches.
WITH ranked AS (
  SELECT p.id, p.slug, p.title, p.summary, p.published_at, p.body_md, p.search_vector, q.query,
    -- Normalization 1 divides by 1 + log(length): a long post that mentions a
    -- term in passing ranks below a short post about it, without burying long
    -- posts the way 2 (divide by length) does.
    ts_rank_cd(p.search_vector, q.query, 1) AS rank
  FROM posts p,
    websearch_to_tsquery('english', sqlc.arg(query)) AS q(query)
  WHERE p.published_at <= CURRENT_TIMESTAMP
    AND p.deleted_at IS NULL
    AND q.query @@ p.search_vector
  ORDER BY rank DESC, p.published_at DESC, p.id DESC
  LIMIT sqlc.arg(result_limit)::int
)
-- ts_headline re-parses each body, so it runs only on the rows that survive
-- the LIMIT. The markers are search.matchStart and search.matchStop.
SELECT id, slug, title, summary, published_at,
  -- Weight C is the body (see the posts migration). When only the title or
  -- summary matched, a headline would be the body's first few words with no
  -- highlight, so return '' and let the client show the summary.
  CASE WHEN ts_filter(search_vector, '{c}') @@ query THEN
    ts_headline('english', body_md, query,
      'StartSel=' || U&'\E000' || ', StopSel=' || U&'\E001' || ', MaxFragments=2, MaxWords=15, MinWords=5'
    )
    ELSE ''
  END::text AS snippet
FROM ranked
ORDER BY rank DESC, published_at DESC, id DESC;
