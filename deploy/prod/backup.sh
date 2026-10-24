#!/usr/bin/env bash
# Dump the blog database and upload it to the private R2 bucket.
# Objects are named blog-YYYYMMDD.dump. Ones dated before 14 days ago are deleted.
# Run from /srv/blog, nightly, by blog-backup.timer.

set -euo pipefail
cd "$(dirname "$0")"

set -a
# shellcheck disable=SC1091
. ./backup.env
set +a

: "${S3_ENDPOINT:?S3_ENDPOINT is not set}"
: "${S3_BUCKET:?S3_BUCKET is not set}"
: "${S3_ACCESS_KEY:?S3_ACCESS_KEY is not set}"
: "${S3_SECRET_KEY:?S3_SECRET_KEY is not set}"

export AWS_ACCESS_KEY_ID="$S3_ACCESS_KEY"
export AWS_SECRET_ACCESS_KEY="$S3_SECRET_KEY"
export AWS_DEFAULT_REGION="${S3_REGION:-auto}"
export AWS_EC2_METADATA_DISABLED=true

# The image entrypoint is the aws CLI. --endpoint-url belongs before the subcommand.
r2() {
	docker run --rm -i \
		-e AWS_ACCESS_KEY_ID \
		-e AWS_SECRET_ACCESS_KEY \
		-e AWS_DEFAULT_REGION \
		-e AWS_EC2_METADATA_DISABLED \
		amazon/aws-cli:2.27.50 \
		--endpoint-url "$S3_ENDPOINT" \
		"$@"
}

# pg_dump must finish before anything is uploaded. Piped into aws s3 cp, a
# dump that dies partway still becomes today's object, and pipefail fails
# the script only after that object has replaced a good dump. The table of
# contents is at the front of a custom-format archive, so pg_restore --list
# accepts one that died while writing data. Reading it through does not.
stamp=$(date -u +%Y%m%d)
dump=$(mktemp)
trap 'rm -f "$dump"' EXIT
docker compose exec -T postgres pg_dump -U blog -Fc --no-owner --no-acl blog >"$dump"
docker compose exec -T postgres pg_restore --file=/dev/null <"$dump"
r2 s3 cp - "s3://${S3_BUCKET}/blog-${stamp}.dump" <"$dump"

cutoff=$(date -u -d '14 days ago' +%Y%m%d)
r2 s3api list-objects-v2 \
	--bucket "$S3_BUCKET" \
	--prefix blog- \
	--query 'Contents[].Key' \
	--output text |
	tr '\t' '\n' |
	while read -r object; do
		[ -n "$object" ] && [ "$object" != "None" ] || continue
		day=${object#blog-}
		day=${day%.dump}
		case $day in
		[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]) ;;
		*) continue ;;
		esac
		if [ "$day" -lt "$cutoff" ]; then
			r2 s3 rm "s3://${S3_BUCKET}/${object}"
		fi
	done
