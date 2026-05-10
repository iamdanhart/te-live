#!/usr/bin/env bash
set -euo pipefail

VENDOR_FILE="router/static/vendor/htmx.min.js"

STATUS=$(dev_tools/check-htmx.sh)
echo "$STATUS"

if [ "$(echo "$STATUS" | jq -r '.up_to_date')" = "true" ]; then
  echo "Already up to date, nothing to do."
  exit 0
fi

VERSION=$(echo "$STATUS" | jq -r '.latest')
URL="https://unpkg.com/htmx.org@${VERSION}/dist/htmx.min.js"

echo "Downloading htmx $VERSION..."
curl -sf "$URL" -o "$VENDOR_FILE"
echo "Updated $VENDOR_FILE to $VERSION"