#!/usr/bin/env bash
# NOT AUDITED — AI-generated tooling. Review before executing in any privileged context.
#
#
# fetch-issue-history.sh
#
# Fetches issues from a GitHub repository and writes them to a JSONL file
# (one JSON object per line). Focused on capturing context: the issue body,
# comments, labels, and linked PRs.
#
# Requirements:
#   - gh  (GitHub CLI, authenticated)
#   - jq
#
# Usage:
#   ./fetch-issue-history.sh [options]
#
# Options:
#   -r, --repo OWNER/REPO   GitHub repository (default: gnolang/gno)
#   -o, --output FILE        Output file path (default: issue-history.jsonl)
#   -l, --limit N            Max number of issues to fetch (default: all)
#   -s, --state STATE        Issue state: open | closed | all (default: all)
#   --since DATE             Only issues created/updated after this date (YYYY-MM-DD)
#   --labels LABELS          Comma-separated labels to filter by
#   --batch-size N           Issues per API page (default: 100, max 100)
#   --with-comments          Also fetch comments for each issue (slow)
#   --with-linked-prs        Also fetch PRs that close/reference each issue
#   --for-digest             Shorthand for --with-comments --with-linked-prs --summary
#   --summary                Print a short summary to stdout when done
#   -h, --help               Show this help message
#
# Output format (JSONL — one JSON object per line):
#   {
#     "number": 1234,
#     "title": "bug: something is broken",
#     "state": "OPEN",
#     "author": "alice",
#     "created_at": "2024-01-15T10:00:00Z",
#     "closed_at": null,
#     "body": "Issue description ...",
#     "labels": ["bug", "gnovm"],
#     "assignees": ["bob"],
#     "milestone": "v1.0",
#     "comment_count": 5,
#     "comments": [...],          // only with --with-comments
#     "linked_prs": [...]         // only with --with-linked-prs
#   }

set -euo pipefail

# ── Defaults ──────────────────────────────────────────────────────────────────
REPO="gnolang/gno"
OUTPUT="issue-history.jsonl"
LIMIT=""
STATE="all"
SINCE=""
LABELS=""
BATCH_SIZE=100
WITH_COMMENTS=false
WITH_LINKED_PRS=false
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
    -r|--repo)         REPO="$2";         shift 2 ;;
    -o|--output)       OUTPUT="$2";       shift 2 ;;
    -l|--limit)        LIMIT="$2";        shift 2 ;;
    -s|--state)        STATE="$2";        shift 2 ;;
    --since)           SINCE="$2";        shift 2 ;;
    --labels)          LABELS="$2";       shift 2 ;;
    --batch-size)      BATCH_SIZE="$2";   shift 2 ;;
    --with-comments)   WITH_COMMENTS=true;  shift ;;
    --with-linked-prs) WITH_LINKED_PRS=true; shift ;;
    --for-digest)      WITH_COMMENTS=true; WITH_LINKED_PRS=true; SUMMARY=true; shift ;;
    --summary)         SUMMARY=true;        shift ;;
    -h|--help)         usage ;;
    *)                 die "Unknown option: $1" ;;
  esac
done

require_cmd gh
require_cmd jq

# ── Validate state ────────────────────────────────────────────────────────────
case "$STATE" in
  open|closed|all) ;; # valid
  *) die "Invalid state: $STATE (use open, closed, or all)" ;;
esac

# ── Fetch issues with GraphQL cursor pagination ──────────────────────────────
info "Fetching issues from $REPO (state=$STATE) ..."

TOTAL_FETCHED=0
PAGE=0
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

# Clear / create output file
: > "$OUTPUT"

while true; do
  PAGE=$((PAGE + 1))

  # Build the search query
  SEARCH_QUERY="repo:$REPO is:issue"
  case "$STATE" in
    open)   SEARCH_QUERY+=" is:open" ;;
    closed) SEARCH_QUERY+=" is:closed" ;;
    all)    ;; # no filter
  esac

  if [[ -n "$SINCE" ]]; then
    SEARCH_QUERY+=" created:>=$SINCE"
  fi

  if [[ -n "$LABELS" ]]; then
    IFS=',' read -ra LABEL_ARR <<< "$LABELS"
    for lbl in "${LABEL_ARR[@]}"; do
      SEARCH_QUERY+=" label:\"$lbl\""
    done
  fi

  # Cursor-based pagination
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
        ... on Issue {
          number
          title
          state
          author { login }
          createdAt
          closedAt
          body
          labels(first: 20) { nodes { name } }
          assignees(first: 10) { nodes { login } }
          milestone { title }
          comments { totalCount }
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
    closed_at: .closedAt,
    body: .body,
    labels: [.labels.nodes[].name],
    assignees: [.assignees.nodes[].login],
    milestone: (.milestone.title // null),
    comment_count: .comments.totalCount
  }' "$RESULT_FILE" >> "$OUTPUT"

  BATCH_COUNT=$(jq '.data.search.nodes | length' "$RESULT_FILE")
  TOTAL_FETCHED=$((TOTAL_FETCHED + BATCH_COUNT))

  info "  page $PAGE: fetched $BATCH_COUNT issues (total: $TOTAL_FETCHED)"

  # Check limit
  if [[ -n "$LIMIT" && "$TOTAL_FETCHED" -ge "$LIMIT" ]]; then
    head -n "$LIMIT" "$OUTPUT" > "$TEMP_DIR/trimmed.jsonl"
    mv "$TEMP_DIR/trimmed.jsonl" "$OUTPUT"
    TOTAL_FETCHED="$LIMIT"
    info "  reached limit of $LIMIT issues, stopping."
    break
  fi

  if [[ "$HAS_NEXT" != "true" ]]; then
    break
  fi

  # Small delay to stay well within rate limits
  sleep 0.3
done

info "Fetched $TOTAL_FETCHED issues total."

# ── Optional enrichments (comments, linked PRs) ──────────────────────────────

enrich_issue() {
  local issue_number="$1"
  local issue_json="$2"
  local enriched="$issue_json"
  local tmp_enrich="$TEMP_DIR/enrich_tmp.json"

  if [[ "$WITH_COMMENTS" == true ]]; then
    if gh api "repos/$REPO/issues/$issue_number/comments" --paginate \
      --jq '[.[] | {user: .user.login, body: .body, created_at: .created_at}]' > "$tmp_enrich" 2>/dev/null; then
      # --paginate may produce multiple JSON arrays; merge them
      enriched=$(echo "$enriched" | jq -c --slurpfile comments <(jq -s 'add // []' "$tmp_enrich") '. + {comments: $comments[0]}')
    else
      enriched=$(echo "$enriched" | jq -c '. + {comments: []}')
    fi
  fi

  if [[ "$WITH_LINKED_PRS" == true ]]; then
    # Use the timeline API to find cross-referenced PRs
    if gh api "repos/$REPO/issues/$issue_number/timeline" --paginate \
      -H "Accept: application/vnd.github.mockingbird-preview+json" \
      --jq '[.[] | select(.event == "cross-referenced" and .source.issue.pull_request != null) | {number: .source.issue.number, title: .source.issue.title, state: .source.issue.state, author: .source.issue.user.login}]' > "$tmp_enrich" 2>/dev/null; then
      enriched=$(echo "$enriched" | jq -c --slurpfile prs <(jq -s 'add // [] | unique_by(.number)' "$tmp_enrich") '. + {linked_prs: $prs[0]}')
    else
      enriched=$(echo "$enriched" | jq -c '. + {linked_prs: []}')
    fi
  fi

  echo "$enriched"
}

if [[ "$WITH_COMMENTS" == true || "$WITH_LINKED_PRS" == true ]]; then
  info "Enriching issues with extra data (this may take a while) ..."

  ENRICHED_FILE="$TEMP_DIR/enriched.jsonl"
  : > "$ENRICHED_FILE"

  DONE=0
  while IFS= read -r line; do
    issue_number=$(echo "$line" | jq -r '.number')
    DONE=$((DONE + 1))

    if (( DONE % 10 == 0 )); then
      info "  enriched $DONE / $TOTAL_FETCHED issues ..."
    fi

    enrich_issue "$issue_number" "$line" >> "$ENRICHED_FILE"

    # Rate-limit protection
    sleep 0.5
  done < "$OUTPUT"

  mv "$ENRICHED_FILE" "$OUTPUT"
  info "Enrichment complete."
fi

# ── Summary ───────────────────────────────────────────────────────────────────
if [[ "$SUMMARY" == true ]]; then
  echo ""
  echo "=== Issue Fetch Summary ==="
  echo "Repository:    $REPO"
  echo "State filter:  $STATE"
  echo "Total issues:  $TOTAL_FETCHED"
  echo "Output file:   $OUTPUT"
  echo "==========================="
fi

info "Done. Output written to: $OUTPUT"
