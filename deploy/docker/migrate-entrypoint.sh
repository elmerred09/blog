#!/bin/sh
# goose reads DRIVER and DBSTRING as arguments. Compose passes the goose
# command ("up") and the API env file only has DATABASE_URL.
set -eu
if [ -z "${DATABASE_URL:-}" ]; then
	echo "DATABASE_URL is not set" >&2
	exit 1
fi
exec /usr/local/bin/goose -dir /migrations postgres "$DATABASE_URL" "$@"
