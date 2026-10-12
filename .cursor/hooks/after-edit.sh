#!/bin/bash

# after-edit.sh - Format Go and web files, and regenerate sqlc output and the
# web API types, after agent edits.

input=$(cat)

file=$(printf '%s' "$input" | sed -n 's/.*"file_path"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)
[ -z "$file" ] && exit 0

case "$file" in
  */internal/db/* | */web/lib/api/schema.d.ts | */web/node_modules/* | */web/.next/*)
    # Generated code, never touch it.
    ;;
  *.go)
    make --no-print-directory fmt FILES="$file" >&2
    ;;
  */db/queries/*.sql | */db/migrations/*.sql)
    make --no-print-directory sqlc >&2
    ;;
  */api/openapi.yaml)
    make --no-print-directory web-types >&2
    ;;
  */web/*)
    make --no-print-directory web-fmt WEB_FILES="$file" >&2
    ;;
esac

exit 0
