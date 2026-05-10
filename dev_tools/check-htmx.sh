#!/usr/bin/env bash
set -euo pipefail

VENDOR_FILE="router/static/vendor/htmx.min.js"

LATEST=$(curl -sf "https://api.github.com/repos/bigskysoftware/htmx/releases" \
  | jq -r '[.[] | select(.tag_name | test("^v[0-9]+\\.[0-9]+\\.[0-9]+$"))] | first | .tag_name')

if [ -z "$LATEST" ]; then
  echo "Failed to fetch latest htmx version" >&2
  exit 1
fi

CURRENT=$(grep -o 'version:"[^"]*"' "$VENDOR_FILE" | grep -o '[0-9][^"]*')
LATEST="${LATEST#v}"
UP_TO_DATE=$([ "$CURRENT" = "$LATEST" ] && echo true || echo false)

jq -n --arg current "$CURRENT" --arg latest "$LATEST" --argjson up_to_date "$UP_TO_DATE" \
  '{current: $current, latest: $latest, up_to_date: $up_to_date}'