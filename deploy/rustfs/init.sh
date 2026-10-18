#!/bin/sh
# Creates the blog bucket and applies the public-read and CORS files beside
# this script. Safe to run again: an existing bucket is left in place and the
# policy and CORS are put either way. The bucket and credentials come from
# local.env, which compose.yaml passes in.
set -eu

endpoint="${RUSTFS_URL:-http://rustfs:9000}"
bucket="$S3_BUCKET"
export AWS_ACCESS_KEY_ID="$RUSTFS_ACCESS_KEY"
export AWS_SECRET_ACCESS_KEY="$RUSTFS_SECRET_KEY"
export AWS_DEFAULT_REGION="$S3_REGION"

# RustFS serves path-style URLs (http://host/bucket/key), not virtual hosts.
aws configure set default.s3.addressing_style path

if aws --endpoint-url "$endpoint" s3api head-bucket --bucket "$bucket" >/dev/null 2>&1; then
  echo "bucket $bucket already exists"
else
  aws --endpoint-url "$endpoint" s3 mb "s3://$bucket"
fi

# The policy names the bucket in its ARN; BUCKET is the placeholder.
sed "s/BUCKET/$bucket/" /policy.json > /tmp/policy.json
aws --endpoint-url "$endpoint" s3api put-bucket-policy \
  --bucket "$bucket" \
  --policy file:///tmp/policy.json
aws --endpoint-url "$endpoint" s3api put-bucket-cors \
  --bucket "$bucket" \
  --cors-configuration file:///cors.json
