---
name: explain-query
description: Run EXPLAIN (ANALYZE, BUFFERS) for a SQL query against the local compose Postgres and explain the plan node by node, including estimate vs actual rows, buffer use, and index choices. Use when the user asks why a query is slow, wants a query plan explained, or is checking whether an index is used.
---

# Explain Query

This is a Postgres learning tool. Explain the plan and teach; don't rewrite the user's query or propose the final fix unless they ask. Prefer ending with questions that make them reason about the plan.

## Run it

Start the database if needed:

```bash
docker compose up -d --wait postgres
```

Run the plan through the container so the client matches the server version:

```bash
docker compose exec -T postgres psql -U blog -d blog -c "EXPLAIN (ANALYZE, BUFFERS) <query>"
```

Rules:
- `ANALYZE` **executes** the query. For `INSERT`, `UPDATE`, `DELETE`, wrap it so nothing persists:
  `BEGIN; EXPLAIN (ANALYZE, BUFFERS) ...; ROLLBACK;`
- Replace sqlc parameters (`$1`, `sqlc.arg(...)`) with realistic literals. The chosen plan depends on the values.
- Run it twice and read the second run. The first mostly measures a cold cache (`read` instead of `hit`).
- Tiny tables often get a sequential scan even with an index, because it is cheaper. Check row counts before blaming the index; the plan at 10 vs 10k rows is a good comparison. Use `SET enable_seqscan = off;` only as a labeled experiment, never as a fix.

## Explain the plan

Read from the innermost (most indented) node outward. For each node, cover:

1. **What it does** (Seq Scan, Index Scan, Bitmap Heap/Index Scan, Nested Loop, Hash Join, Sort, Limit, ...) and why the planner likely chose it.
2. **Estimate vs actual:** `rows=` estimated against `actual ... rows=`. A gap of about 10x or more suggests stale or missing statistics (`ANALYZE table`) or correlated columns.
3. **Loops:** actual time and rows are per loop; multiply by `loops`.
4. **Buffers:** `shared hit` (cache) vs `read` (disk). Large `read` counts point to I/O cost.
5. **Waste:** `Rows Removed by Filter` or `by Index Recheck` (an index that isn't selective), Sort spilling to disk (`Sort Method: external merge`), lossy bitmaps.

Then summarize:
- Where most of the time goes, using the top node's `Execution Time` and `Planning Time`.
- What the plan says about indexes: used, unused, or missing.

For the search queries in this project, also check that the GIN index on `search_vector` is used, and that `ts_headline` is applied only to the top N rows, not the whole result set.

## Finish with questions, not answers

End with 2 or 3 questions for the user, such as "Why did the planner pick a Bitmap Heap Scan here instead of an Index Scan?" or "What would you expect to change at 10x the rows?".
