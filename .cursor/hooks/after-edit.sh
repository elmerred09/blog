#!/bin/bash

# after-edit.sh - Format Go files and regenerate sqlc after agent edits.

input=$(cat)

file=$(printf '%s' "$input" | sed -n 's/.*"file_path"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)
[ -z "$file" ] && exit 0

case "$file" in
  */internal/db/*)
    # Generated code, never touch it.
    ;;
  *.go)
    make --no-print-directory fmt FILES="$file" >&2
    ;;
  */db/queries/*.sql | */db/migrations/*.sql)
    make --no-print-directory sqlc >&2
    ;;
esac

exit 0
