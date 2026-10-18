#!/bin/sh
# Creates the blog bucket and applies the public-read and CORS files beside
# this script. Safe to run again: an existing bucket is left in place and the
# policy and CORS are put either way.
set -eu

endpoint="${RUSTFS_URL:-http://rustfs:9000}"
bucket=blog

# RustFS serves path-style URLs (http://host/bucket/key), not virtual hosts.
aws configure set default.s3.addressing_style path

if aws --endpoint-url "$endpoint" s3api head-bucket --bucket "$bucket" >/dev/null 2>&1; then
  echo "bucket $bucket already exists"
else
  aws --endpoint-url "$endpoint" s3 mb "s3://$bucket"
fi

aws --endpoint-url "$endpoint" s3api put-bucket-policy \
  --bucket "$bucket" \
  --policy file:///policy.json
aws --endpoint-url "$endpoint" s3api put-bucket-cors \
  --bucket "$bucket" \
  --cors-configuration file:///cors.json
