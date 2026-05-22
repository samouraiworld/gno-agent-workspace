#!/usr/bin/env bash
# NOT AUDITED — AI-generated tooling. Review before executing in any privileged context.
#
#
# fetch-pr-history.sh
#
# Fetches the full merged PR history from gnolang/gno and writes it to a
# structured JSONL file (one JSON object per line) that can be fed to an LLM
# to understand the project's evolution.
#
# Requirements:
#   - gh  (GitHub CLI, authenticated)
#   - jq
#
# Usage:
#   ./fetch-pr-history.sh [options]
#
# Options:
#   -r, --repo OWNER/REPO   GitHub repository (default: gnolang/gno)
#   -o, --output FILE        Output file path (default: pr-history.jsonl)
#   -l, --limit N            Max number of PRs to fetch (default: all)
#   -s, --state STATE        PR state: merged | closed | open | all (default: merged)
#   --since DATE             Only PRs merged/updated after this date (YYYY-MM-DD)
#   --labels LABELS          Comma-separated labels to filter by
#   --batch-size N           PRs per API page (default: 100, max 100)
#   --with-comments          Also fetch review comments for each PR (slow)
#   --with-files             Also fetch the list of changed files per PR (includes patches)
#   --with-file-paths        Like --with-files but without patch content (lean)
#   --with-reviews           Also fetch review decisions per PR
#   --for-digest             Shorthand for --with-files --with-reviews --with-comments --summary
#   --summary                Print a short summary to stdout when done
#   -h, --help               Show this help message
#
# Output format (JSONL — one JSON object per line):
#   {
#     "number": 1234,
#     "title": "feat: add foo",
#     "state": "MERGED",
#     "author": "alice",
#     "created_at": "2024-01-15T10:00:00Z",
#     "merged_at": "2024-01-16T12:00:00Z",
#     "closed_at": "2024-01-16T12:00:00Z",
#     "body": "PR description ...",
#     "additions": 42,
#     "deletions": 7,
#     "changed_files_count": 3,
#     "comments": [...],          // only with --with-comments
#     "files": [...],             // only with --with-files
#     "reviews": [...]            // only with --with-reviews
#   }

set -euo pipefail

# ── Defaults ──────────────────────────────────────────────────────────────────
REPO="gnolang/gno"
OUTPUT="pr-history.jsonl"
LIMIT=""
STATE="merged"
SINCE=""
LABELS=""
BATCH_SIZE=100
WITH_COMMENTS=false
WITH_FILES=false
WITH_FILE_PATHS=false
WITH_REVIEWS=false
SUMMARY=false

# ── Helpers ───────────────────────────────────────────────────────────────────
die()  { echo "ERROR: $*" >&2; exit 1; }
info() { echo ":: $*" >&2; }

usage() {
  sed -n '/^# Usage:/,/^[^#]/{ /^#/s/^# \?//p }' "$0"
  exit 0
}

require_cmd() {
  command -v "$1" &>/dev/null || die "'$1' is required but not found in PATH"
}

# ── Parse arguments ───────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    -r|--repo)        REPO="$2";        shift 2 ;;
    -o|--output)      OUTPUT="$2";      shift 2 ;;
    -l|--limit)       LIMIT="$2";       shift 2 ;;
    -s|--state)       STATE="$2";       shift 2 ;;
    --since)          SINCE="$2";       shift 2 ;;
    --labels)         LABELS="$2";      shift 2 ;;
    --batch-size)     BATCH_SIZE="$2";  shift 2 ;;
    --with-comments)  WITH_COMMENTS=true; shift ;;
    --with-files)     WITH_FILES=true;    shift ;;
    --with-file-paths) WITH_FILE_PATHS=true; shift ;;
    --with-reviews)   WITH_REVIEWS=true;  shift ;;
    --for-digest)     WITH_FILES=true; WITH_REVIEWS=true; WITH_COMMENTS=true; SUMMARY=true; shift ;;
    --summary)        SUMMARY=true;       shift ;;
    -h|--help)        usage ;;
    *)                die "Unknown option: $1" ;;
  esac
done

require_cmd gh
require_cmd jq

# ── Build the search query ─────────────────────────────────────────────────────

case "$STATE" in
  merged|closed|open|all) ;; # valid
  *) die "Invalid state: $STATE (use merged, closed, open, or all)" ;;
esac

# ── Fetch PRs with pagination ────────────────────────────────────────────────
info "Fetching PRs from $REPO (state=$STATE) ..."

TOTAL_FETCHED=0
PAGE=0
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

# Clear / create output file
: > "$OUTPUT"

while true; do
  PAGE=$((PAGE + 1))

  # gh pr list does not have a native --page flag, but --limit handles
  # pagination implicitly when we use the GraphQL-backed JSON output.
  # We use `gh api` with GraphQL for reliable cursor-based pagination.

  # Build the search query for gh api
  SEARCH_QUERY="repo:$REPO is:pr"
  case "$STATE" in
    merged) SEARCH_QUERY+=" is:merged" ;;
    closed) SEARCH_QUERY+=" is:closed is:unmerged" ;;
    open)   SEARCH_QUERY+=" is:open" ;;
    all)    ;; # no filter
  esac

  if [[ -n "$SINCE" ]]; then
    SEARCH_QUERY+=" merged:>=$SINCE"
  fi

  if [[ -n "$LABELS" ]]; then
    IFS=',' read -ra LABEL_ARR <<< "$LABELS"
    for lbl in "${LABEL_ARR[@]}"; do
      SEARCH_QUERY+=" label:\"$lbl\""
    done
  fi

  # Use GraphQL for reliable cursor-based pagination
  AFTER_CLAUSE=""
  if [[ -n "${CURSOR:-}" ]]; then
    AFTER_CLAUSE=", after: \"$CURSOR\""
  fi

  GRAPHQL_QUERY="
  {
    search(query: \"$SEARCH_QUERY\", type: ISSUE, first: $BATCH_SIZE$AFTER_CLAUSE) {
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        ... on PullRequest {
          number
          title
          state
          author { login }
          createdAt
          mergedAt
          closedAt
          body
          additions
          deletions
          changedFiles
        }
      }
    }
  }"

  RESULT_FILE="$TEMP_DIR/page_${PAGE}.json"

  if ! gh api graphql -f query="$GRAPHQL_QUERY" > "$RESULT_FILE" 2>/dev/null; then
    die "GraphQL query failed on page $PAGE. Check your 'gh' auth and network."
  fi

  # Extract nodes and pagination info
  HAS_NEXT=$(jq -r '.data.search.pageInfo.hasNextPage' "$RESULT_FILE")
  CURSOR=$(jq -r '.data.search.pageInfo.endCursor' "$RESULT_FILE")
  NODE_COUNT=$(jq '.data.search.nodes | length' "$RESULT_FILE")

  if [[ "$NODE_COUNT" -eq 0 ]]; then
    break
  fi

  # Transform each node into our output format and append to JSONL
  jq -c '.data.search.nodes[] | {
    number: .number,
    title: .title,
    state: .state,
    author: (.author.login // "ghost"),
    created_at: .createdAt,
    merged_at: .mergedAt,
    closed_at: .closedAt,
    body: .body,
    additions: .additions,
    deletions: .deletions,
    changed_files_count: .changedFiles
  }' "$RESULT_FILE" >> "$OUTPUT"

  BATCH_COUNT=$(jq '.data.search.nodes | length' "$RESULT_FILE")
  TOTAL_FETCHED=$((TOTAL_FETCHED + BATCH_COUNT))

  info "  page $PAGE: fetched $BATCH_COUNT PRs (total: $TOTAL_FETCHED)"

  # Check limit
  if [[ -n "$LIMIT" && "$TOTAL_FETCHED" -ge "$LIMIT" ]]; then
    # Trim output to exactly LIMIT lines
    head -n "$LIMIT" "$OUTPUT" > "$TEMP_DIR/trimmed.jsonl"
    mv "$TEMP_DIR/trimmed.jsonl" "$OUTPUT"
    TOTAL_FETCHED="$LIMIT"
    info "  reached limit of $LIMIT PRs, stopping."
    break
  fi

  if [[ "$HAS_NEXT" != "true" ]]; then
    break
  fi

  # Small delay to stay well within rate limits
  sleep 0.3
done

info "Fetched $TOTAL_FETCHED PRs total."

# ── Optional enrichments (comments, files, reviews) ──────────────────────────
# These require per-PR API calls, so they are opt-in.

enrich_pr() {
  local pr_number="$1"
  local pr_json="$2"
  local enriched="$pr_json"
  local tmp_enrich="$TEMP_DIR/enrich_tmp.json"

  if [[ "$WITH_FILES" == true ]]; then
    if gh api "repos/$REPO/pulls/$pr_number/files" --paginate \
      --jq '[.[] | {filename: .filename, status: .status, additions: .additions, deletions: .deletions, patch: .patch}]' > "$tmp_enrich" 2>/dev/null; then
      # --paginate may produce multiple JSON arrays; merge them
      enriched=$(echo "$enriched" | jq -c --slurpfile files <(jq -s 'add' "$tmp_enrich") '. + {files: $files[0]}')
    else
      enriched=$(echo "$enriched" | jq -c '. + {files: []}')
    fi
  elif [[ "$WITH_FILE_PATHS" == true ]]; then
    if gh api "repos/$REPO/pulls/$pr_number/files" --paginate \
      --jq '[.[] | {filename: .filename, status: .status, additions: .additions, deletions: .deletions}]' > "$tmp_enrich" 2>/dev/null; then
      enriched=$(echo "$enriched" | jq -c --slurpfile files <(jq -s 'add' "$tmp_enrich") '. + {files: $files[0]}')
    else
      enriched=$(echo "$enriched" | jq -c '. + {files: []}')
    fi
  fi

  if [[ "$WITH_REVIEWS" == true ]]; then
    if gh api "repos/$REPO/pulls/$pr_number/reviews" --paginate \
      --jq '[.[] | {user: .user.login, state: .state, body: .body, submitted_at: .submitted_at}]' > "$tmp_enrich" 2>/dev/null; then
      enriched=$(echo "$enriched" | jq -c --slurpfile reviews <(jq -s 'add' "$tmp_enrich") '. + {reviews: $reviews[0]}')
    else
      enriched=$(echo "$enriched" | jq -c '. + {reviews: []}')
    fi
  fi

  if [[ "$WITH_COMMENTS" == true ]]; then
    if gh api "repos/$REPO/pulls/$pr_number/comments" --paginate \
      --jq '[.[] | {user: .user.login, body: .body, created_at: .created_at, path: .path, line: .line}]' > "$tmp_enrich" 2>/dev/null; then
      enriched=$(echo "$enriched" | jq -c --slurpfile comments <(jq -s 'add' "$tmp_enrich") '. + {comments: $comments[0]}')
    else
      enriched=$(echo "$enriched" | jq -c '. + {comments: []}')
    fi
  fi

  echo "$enriched"
}

if [[ "$WITH_COMMENTS" == true || "$WITH_FILES" == true || "$WITH_FILE_PATHS" == true || "$WITH_REVIEWS" == true ]]; then
  info "Enriching PRs with extra data (this may take a while) ..."

  ENRICHED_FILE="$TEMP_DIR/enriched.jsonl"
  : > "$ENRICHED_FILE"

  DONE=0
  while IFS= read -r line; do
    pr_number=$(echo "$line" | jq -r '.number')
    DONE=$((DONE + 1))

    if (( DONE % 10 == 0 )); then
      info "  enriched $DONE / $TOTAL_FETCHED PRs ..."
    fi

    enrich_pr "$pr_number" "$line" >> "$ENRICHED_FILE"

    # Rate-limit protection
    sleep 0.5
  done < "$OUTPUT"

  mv "$ENRICHED_FILE" "$OUTPUT"
  info "Enrichment complete."
fi

# ── Summary ───────────────────────────────────────────────────────────────────
if [[ "$SUMMARY" == true ]]; then
  echo ""
  echo "=== PR History Summary ==="
  echo "Repository:  $REPO"
  echo "State:       $STATE"
  echo "Total PRs:   $TOTAL_FETCHED"
  echo "Output file: $OUTPUT"
  echo ""

  if [[ "$TOTAL_FETCHED" -gt 0 ]]; then
    echo "Top 10 contributors:"
    jq -r '.author' "$OUTPUT" | sort | uniq -c | sort -rn | head -10 | \
      awk '{ printf "  %4d PRs  %s\n", $1, $2 }'
    echo ""

    echo "PRs by year:"
    jq -r '.created_at // empty' "$OUTPUT" | cut -c1-4 | sort | uniq -c | sort -k2 | \
      awk '{ printf "  %s: %d PRs\n", $2, $1 }'
    echo ""

    total_additions=$(jq -s '[.[].additions // 0] | add' "$OUTPUT")
    total_deletions=$(jq -s '[.[].deletions // 0] | add' "$OUTPUT")
    echo "Total lines added:   $total_additions"
    echo "Total lines deleted: $total_deletions"
  fi

  echo "========================="
fi

info "Done. Output written to: $OUTPUT"
