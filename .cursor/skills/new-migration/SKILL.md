---
name: new-migration
description: Create a goose SQL migration for the blog's Postgres schema, verify it applies and reverts against the local compose database, then regenerate sqlc. Use when adding or changing tables, columns, indexes, or constraints, or when the user asks for a new migration.
---

# New Migration

Migrations live in `db/migrations` and are append-only (see `.cursor/rules/sql.mdc`). Never edit a migration that has been committed.

SQL is a learning goal for this project: create the file skeleton, but let the user write the schema SQL unless they ask you to draft it.

## Workflow

```
- [ ] 1. Start Postgres
- [ ] 2. Create the migration file
- [ ] 3. Fill in Up and Down
- [ ] 4. Verify up, down, up
- [ ] 5. Regenerate sqlc
```

**1. Start Postgres** (waits until healthy):

```bash
docker compose up -d --wait postgres
```

**2. Create the file.** Use a short snake_case name describing the change:

```bash
GOOSE_DRIVER=postgres GOOSE_DBSTRING="postgres://blog:blog@localhost:5432/blog?sslmode=disable" \
  go tool goose -dir db/migrations create add_posts_table sql
```

**3. Fill in both sections.** The Down must fully reverse the Up:

```sql
-- +goose Up
CREATE TABLE tags (
    id   bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name text NOT NULL UNIQUE
);

-- +goose Down
DROP TABLE tags;
```

For statements containing `;` inside a body (functions, DO blocks), wrap them in `-- +goose StatementBegin` / `-- +goose StatementEnd`.

**4. Verify the round trip.** Set the env once, then run:

```bash
export GOOSE_DRIVER=postgres GOOSE_MIGRATION_DIR=db/migrations
export GOOSE_DBSTRING="postgres://blog:blog@localhost:5432/blog?sslmode=disable"
go tool goose up
go tool goose down    # reverts only the latest migration
go tool goose up
go tool goose status
```

If `down` fails or leaves objects behind, fix the Down section before continuing. If you must change an Up that was already applied locally, run `go tool goose down` first.

**5. Regenerate sqlc** and confirm it is clean:

```bash
make sqlc
make sqlc-check
```

These skip with a message until `sqlc.yaml` exists. If the migration changes a column that existing queries in `db/queries` use, update those queries too.

## Checks before finishing

- Both `Up` and `Down` are present and the round trip passed.
- New query patterns have a matching index, or a comment explaining why not.
- `make lint` and `make sqlc-check` pass.
