#!/usr/bin/env bash
# Parse context.md for AI consumption: strip titles and auto-detected statuses.
# The AI already has PR status from the JSON data — only manual notes matter.
# Usage: ./scripts/parse-context.sh reports/weekly/2026-04-12/context.md

set -euo pipefail

file="${1:?Usage: $0 <context.md path>}"

if [[ ! -f "$file" ]]; then
  echo "Error: file not found: $file" >&2
  exit 1
fi

sed \
  -e 's/ - `.*$//' \
  -e 's/Approved//' \
  -e 's/Changes requested//' \
  -e 's/In progress//' \
  -e "s/Don't merge//" \
  -e 's/Waiting for first review//' \
  -e 's/,[[:space:]]*$//' \
  -e 's/[[:space:]]*$//' \
  "$file"
