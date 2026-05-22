#!/usr/bin/env bash
# NOT AUDITED — AI-generated tooling. Review before executing in any privileged context.
#
#
# weekly-report.sh
#
# Gathers PR data for the Samourai team's weekly report. Fetches open and
# recently merged PRs authored by team members across configured repos,
# then outputs a single JSON file the AI uses to draft the report.
#
# Requirements:
#   - gh  (GitHub CLI, authenticated)
#   - jq
#
# Usage:
#   ./scripts/weekly-report.sh [options]
#
# Options:
#   -o, --output FILE       Output file (default: data/weekly-report-data.json)
#   -d, --days N            Look-back window for merged PRs (default: 7)
#   -e, --end-date DATE     End date of the report period (YYYY-MM-DD, default: today)
#   -h, --help              Show this help
#
# The script reads TEAM_MEMBERS and REPOS from the environment or uses defaults.
# Override with space-separated values:
#   TEAM_MEMBERS="davd-gzl omarsy" REPOS="gnolang/gno" ./scripts/weekly-report.sh

set -euo pipefail

# ── Defaults ──────────────────────────────────────────────────────────────────
OUTPUT="data/weekly-report-data.json"
DAYS=7
END_DATE=""

# Team GitHub usernames (override via env)
: "${TEAM_MEMBERS:=davd-gzl omarsy mvallenet Villaquiranm WaDadidou zxxma louis14448 AmozPay}"

# Repos to track (override via env)
: "${REPOS:=gnolang/gno samouraiworld/gnomonitoring}"

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
    -o|--output)   OUTPUT="$2";   shift 2 ;;
    -d|--days)     DAYS="$2";     shift 2 ;;
    -e|--end-date) END_DATE="$2"; shift 2 ;;
    -h|--help)     usage ;;
    *)           die "Unknown option: $1" ;;
  esac
done

require_cmd gh
require_cmd jq

# ── Validate team handles ─────────────────────────────────────────────────────
# GitHub renames silently drop PRs: `gh search` returns empty for unknown
# authors instead of erroring. Fail loudly if any handle no longer exists.
MISSING_HANDLES=()
for member in $TEAM_MEMBERS; do
  gh api "users/$member" --jq .login &>/dev/null || MISSING_HANDLES+=("$member")
done
if (( ${#MISSING_HANDLES[@]} > 0 )); then
  die "Unknown GitHub handle(s): ${MISSING_HANDLES[*]} — update TEAM_MEMBERS in this script and the Samourai list in skills/weekly-report.md."
fi

# ── Setup ─────────────────────────────────────────────────────────────────────
if [[ -n "$END_DATE" ]]; then
  TODAY="$END_DATE"
  SINCE_DATE=$(date -d "$END_DATE - $DAYS days" +%Y-%m-%d 2>/dev/null || date -j -v-${DAYS}d -f "%Y-%m-%d" "$END_DATE" +%Y-%m-%d)
else
  TODAY=$(date +%Y-%m-%d)
  SINCE_DATE=$(date -d "$DAYS days ago" +%Y-%m-%d 2>/dev/null || date -v-${DAYS}d +%Y-%m-%d)
fi

mkdir -p "$(dirname "$OUTPUT")"
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

info "Team members: $TEAM_MEMBERS"
info "Repos: $REPOS"
info "Window: $SINCE_DATE .. $TODAY ($DAYS days)"
info "Output: $OUTPUT"

REPO_INDEX=0

# ── Fetch per repo ───────────────────────────────────────────────────────────
for repo in $REPOS; do
  info "Fetching from $repo ..."

  OPEN_FILE="$TEMP_DIR/open_${REPO_INDEX}.json"
  MERGED_FILE="$TEMP_DIR/merged_${REPO_INDEX}.json"
  echo "[]" > "$OPEN_FILE"
  echo "[]" > "$MERGED_FILE"

  # --- Open PRs by team members ---
  for member in $TEAM_MEMBERS; do
    MEMBER_FILE="$TEMP_DIR/member_open.json"
    gh pr list --repo "$repo" --author "$member" --state open \
      --json number,title,author,labels,createdAt,url,isDraft,reviewRequests,reviewDecision,body,mergeable \
      --limit 200 2>/dev/null \
      | jq '[.[] | {
          number, title, url, createdAt, isDraft, reviewDecision,
          author: .author.login,
          labels: [.labels[].name],
          reviewRequests: [.reviewRequests[] | .login // .slug // .name],
          body, mergeable
        }]' > "$MEMBER_FILE" 2>/dev/null || echo "[]" > "$MEMBER_FILE"
    jq -s '.[0] + .[1]' "$OPEN_FILE" "$MEMBER_FILE" > "$TEMP_DIR/tmp.json" && mv "$TEMP_DIR/tmp.json" "$OPEN_FILE"
  done

  # --- Recently merged PRs by team members ---
  for member in $TEAM_MEMBERS; do
    MEMBER_FILE="$TEMP_DIR/member_merged.json"
    SEARCH_Q="repo:$repo is:pr is:merged author:$member merged:$SINCE_DATE..$TODAY"
    gh api graphql -f query="
    {
      search(query: \"$SEARCH_Q\", type: ISSUE, first: 100) {
        nodes {
          ... on PullRequest {
            number
            title
            url
            author { login }
            mergedAt
            labels(first: 10) { nodes { name } }
          }
        }
      }
    }" --jq '[.data.search.nodes[] | {
      number: .number,
      title: .title,
      url: .url,
      author: (.author.login // "ghost"),
      mergedAt: .mergedAt,
      labels: [.labels.nodes[].name]
    }]' > "$MEMBER_FILE" 2>/dev/null || echo "[]" > "$MEMBER_FILE"
    jq -s '.[0] + .[1]' "$MERGED_FILE" "$MEMBER_FILE" > "$TEMP_DIR/tmp.json" && mv "$TEMP_DIR/tmp.json" "$MERGED_FILE"
  done

  # Deduplicate by PR number
  jq '[group_by(.number)[] | first]' "$OPEN_FILE" > "$TEMP_DIR/tmp.json" && mv "$TEMP_DIR/tmp.json" "$OPEN_FILE"
  jq '[group_by(.number)[] | first]' "$MERGED_FILE" > "$TEMP_DIR/tmp.json" && mv "$TEMP_DIR/tmp.json" "$MERGED_FILE"

  # --- Fetch review stats and mergeable status for ALL open PRs ---
  ALL_PR_NUMBERS=$(jq -r '.[].number' "$OPEN_FILE")
  if [[ -n "$ALL_PR_NUMBERS" ]]; then
    PR_COUNT=$(echo "$ALL_PR_NUMBERS" | wc -l | tr -d ' ')
    info "Fetching review stats + mergeable for $PR_COUNT open PRs in $repo ..."
    ENRICHED_FILE="$TEMP_DIR/enriched_open.json"
    cp "$OPEN_FILE" "$ENRICHED_FILE"

    for pr_num in $ALL_PR_NUMBERS; do
      REVIEW_FILE="$TEMP_DIR/reviews_${pr_num}.json"
      gh pr view "$pr_num" --repo "$repo" --json reviews,mergeable \
        --jq '{reviews: [.reviews[] | {author: .author.login, state: .state}], mergeable: .mergeable}' \
        > "$REVIEW_FILE" 2>/dev/null || echo '{"reviews":[],"mergeable":"UNKNOWN"}' > "$REVIEW_FILE"

      # Compute review stats: last review per author = their official status.
      # Includes approver usernames so AI can check core team vs Samourai.
      STATS=$(jq -r '
        .reviews | group_by(.author) | map(last) |
        {
          approved:          [.[] | select(.state == "APPROVED")]          | length,
          commented:         [.[] | select(.state == "COMMENTED")]         | length,
          changes_requested: [.[] | select(.state == "CHANGES_REQUESTED")] | length,
          approvers:         [.[] | select(.state == "APPROVED") | .author],
          changes_requesters: [.[] | select(.state == "CHANGES_REQUESTED") | .author]
        }
      ' "$REVIEW_FILE")

      MERGEABLE=$(jq -r '.mergeable' "$REVIEW_FILE")

      # Inject reviewStats and accurate mergeable into the matching PR object
      jq --argjson num "$pr_num" --argjson stats "$STATS" --arg merge "$MERGEABLE" \
        '[.[] | if .number == $num then . + {reviewStats: $stats, mergeable: $merge} else . end]' \
        "$ENRICHED_FILE" > "$TEMP_DIR/tmp.json" && mv "$TEMP_DIR/tmp.json" "$ENRICHED_FILE"
    done

    mv "$ENRICHED_FILE" "$OPEN_FILE"
  fi

  # --- Issues opened this week ---
  ISSUES_FILE="$TEMP_DIR/issues_${REPO_INDEX}.json"
  echo "[]" > "$ISSUES_FILE"
  info "Fetching issues opened this week in $repo ..."
  for member in $TEAM_MEMBERS; do
    MEMBER_FILE="$TEMP_DIR/member_issues.json"
    SEARCH_Q="repo:$repo is:issue author:$member created:$SINCE_DATE..$TODAY"
    gh api graphql -f query="
    {
      search(query: \"$SEARCH_Q\", type: ISSUE, first: 100) {
        nodes {
          ... on Issue {
            number
            title
            url
            author { login }
            createdAt
            state
            labels(first: 10) { nodes { name } }
          }
        }
      }
    }" --jq '[.data.search.nodes[] | {
      number: .number,
      title: .title,
      url: .url,
      author: (.author.login // "ghost"),
      createdAt: .createdAt,
      state: .state,
      labels: [.labels.nodes[].name]
    }]' > "$MEMBER_FILE" 2>/dev/null || echo "[]" > "$MEMBER_FILE"
    jq -s '.[0] + .[1]' "$ISSUES_FILE" "$MEMBER_FILE" > "$TEMP_DIR/tmp.json" && mv "$TEMP_DIR/tmp.json" "$ISSUES_FILE"
  done
  # Deduplicate by issue number
  jq '[group_by(.number)[] | first]' "$ISSUES_FILE" > "$TEMP_DIR/tmp.json" && mv "$TEMP_DIR/tmp.json" "$ISSUES_FILE"

  # Build per-repo JSON
  jq -n \
    --arg repo "$repo" \
    --slurpfile open "$OPEN_FILE" \
    --slurpfile merged "$MERGED_FILE" \
    --slurpfile issues "$ISSUES_FILE" \
    '{repo: $repo, open_prs: $open[0], merged_prs: $merged[0], issues_opened: $issues[0]}' > "$TEMP_DIR/repo_${REPO_INDEX}.json"

  REPO_INDEX=$((REPO_INDEX + 1))
done

# ── Assemble final output ────────────────────────────────────────────────────
# Merge all repo JSON files into an array
jq -s '.' "$TEMP_DIR"/repo_*.json > "$TEMP_DIR/all_repos.json"

jq -c -n \
  --arg since "$SINCE_DATE" \
  --arg until "$TODAY" \
  --arg team "$TEAM_MEMBERS" \
  --slurpfile repos "$TEMP_DIR/all_repos.json" \
  '{
    generated_at: now | todate,
    period: {from: $since, to: $until},
    team_members: ($team | split(" ")),
    repos: $repos[0]
  }' > "$OUTPUT"

OPEN_COUNT=$(jq '[.repos[].open_prs | length] | add' "$OUTPUT")
MERGED_COUNT=$(jq '[.repos[].merged_prs | length] | add' "$OUTPUT")
ISSUES_COUNT=$(jq '[.repos[].issues_opened | length] | add' "$OUTPUT")

info "Done. Open PRs: $OPEN_COUNT, Merged PRs: $MERGED_COUNT, Issues opened: $ISSUES_COUNT"
info "Output: $OUTPUT"
