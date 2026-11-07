#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "usage: rekor-v2-config.sh <output.json>" >&2
  exit 1
fi

raw="$(mktemp)"
trap 'rm -f "$raw" "$raw.filtered"' EXIT

# Cosign authenticates this distinct signing configuration through Sigstore TUF.
cosign signing-config create --with-default-rekor-v2 --out "$raw"
jq -e '
  .rekorTlogUrls |= map(select(.majorApiVersion == 2))
  | if (.rekorTlogUrls | length) == 0 then
      error("authenticated signing configuration has no Rekor v2 service")
    else . end
' "$raw" > "$raw.filtered"
mv -- "$raw.filtered" "$1"
