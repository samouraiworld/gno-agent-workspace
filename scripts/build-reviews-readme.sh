#!/usr/bin/env bash
# NOT AUDITED — AI-generated tooling. Review before executing in any privileged context.
#
#
# build-reviews-readme.sh
#
# Regenerates reviews/README.md by scanning reviews/pr/<thousand>xxx/<num>-*/
# and querying gh for each PR's current state. Fast (parallel gh calls) and
# truthful (always pulls live state). Run any time the dir layout changes or
# after a batch of new reviews.
#
# Requirements: gh (authenticated), jq
#
# Usage:
#   ./scripts/build-reviews-readme.sh [-o OUTPUT]

set -euo pipefail

OUTPUT="reviews/README.md"
REPO="gnolang/gno"

# Samourai team handles. Mirror of TEAM_MEMBERS in scripts/weekly-report.sh —
# keep both in sync.
: "${TEAM_MEMBERS:=davd-gzl omarsy mvallenet Villaquiranm WaDadidou zxxma louis14448 AmozPay}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -o|--output) OUTPUT="$2"; shift 2 ;;
    -h|--help)
      sed -n '2,15p' "$0"; exit 0 ;;
    *) echo "unknown arg: $1" >&2; exit 1 ;;
  esac
done

cd "$(git rev-parse --show-toplevel)"

if [[ ! -d reviews/pr ]]; then
  echo "reviews/pr not found" >&2
  exit 1
fi

# Collect (bucket, num, dir) tuples
mapfile -t DIRS < <(find reviews/pr -mindepth 2 -maxdepth 2 -type d | sort)

if [[ ${#DIRS[@]} -eq 0 ]]; then
  echo "no review dirs found under reviews/pr/" >&2
  exit 1
fi

# Fetch PR metadata in parallel into a temp dir (one JSON per PR).
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

fetch_one() {
  local num="$1"
  gh pr view "$num" -R "$REPO" \
    --json number,state,title,mergedAt,closedAt,url,isDraft \
    > "$TMP/$num.json" 2>/dev/null \
    || echo "{\"number\":$num,\"state\":\"UNKNOWN\",\"title\":\"\",\"url\":\"https://github.com/$REPO/pull/$num\"}" > "$TMP/$num.json"
}

PR_NUMS=()
for d in "${DIRS[@]}"; do
  base=$(basename "$d")
  num="${base%%-*}"
  [[ "$num" =~ ^[0-9]+$ ]] || continue
  PR_NUMS+=("$num")
done

echo "Fetching ${#PR_NUMS[@]} PRs from $REPO..." >&2

# Run up to 8 in parallel
JOBS=8
i=0
for num in "${PR_NUMS[@]}"; do
  fetch_one "$num" &
  i=$((i+1))
  if (( i % JOBS == 0 )); then wait; fi
done
wait

# Fetch every open PR with its reviews + comments in one shot for the team
# coverage section below. We do this here (parallel to per-review fetches above)
# so we pay one round-trip even though it's a larger payload.
echo "Fetching open PRs + reviews for team coverage section..." >&2
gh pr list -R "$REPO" --state open --limit 300 \
  --json number,title,url,isDraft,author,createdAt,updatedAt,reviews,comments \
  > "$TMP/open-prs.json" 2>/dev/null || echo "[]" > "$TMP/open-prs.json"

# Build markdown body to a temp file first so we can generate a TOC from its
# real H2/H3 headings, then assemble header + TOC + body into the final file.
BODY="$TMP/body.md"
{
  # ── Team coverage on open PRs ───────────────────────────────────────────────
  # For each non-draft open PR, surface the Samourai team's latest review state.
  # Aggregation: any CHANGES_REQUESTED → 🔴; else any APPROVED → 🟢; else any
  # COMMENTED review or any issue comment by a team member → 💬; else ⏳.
  team_count=$(jq --arg team "$TEAM_MEMBERS" '
    ($team | split(" ")) as $T
    | map(select(.isDraft | not))
    | length
  ' "$TMP/open-prs.json")

  echo "## Team coverage on open PRs ($team_count)"
  echo
  echo "Triage view. Samourai handles: $(echo "$TEAM_MEMBERS" | sed 's/ /, /g'). Sorted with ⏳ first, then 💬, then 🔴, then 🟢; ties broken by most-recently-updated."
  echo
  echo "| PR | Title | Author | Team coverage | AI review |"
  echo "|---:|:------|:-------|:--------------|:-------------|"

  # Build local-review dir + verdict + model maps, keyed by PR number.
  # Verdict aggregation across the latest round's review files:
  #   REQUEST CHANGES > NEEDS DISCUSSION > APPROVE; ⚪ no verdict if nothing extractable.
  # Model: every unique <model> parsed from `<model>_<reviewer>.md` filenames,
  # `claude-` prefix stripped for compactness.
  declare -A LOCAL_REVIEW
  declare -A LOCAL_VERDICT
  declare -A LOCAL_MODELS
  while IFS= read -r d; do
    base=$(basename "$d")
    n="${base%%-*}"
    [[ "$n" =~ ^[0-9]+$ ]] || continue
    # Latest round dir: <n>-<hash>, sorted by leading round number.
    latest_round=$(find "$d" -mindepth 1 -maxdepth 1 -type d -regextype posix-extended -regex '.*/[0-9]+-[a-f0-9]+' \
      | awk -F/ '{print $NF, $0}' \
      | sort -k1,1n \
      | tail -1 \
      | awk '{print $2}')
    if [[ -n "$latest_round" ]]; then
      # Link to the latest round (its dir name encodes the commit hash).
      LOCAL_REVIEW[$n]="${latest_round#reviews/}"
    else
      LOCAL_REVIEW[$n]="${d#reviews/}"
      continue
    fi

    # Aggregate verdict + collect models across every reviewer file in the latest round.
    # Verdict line may take any of these forms (in priority order):
    #   **Verdict: APPROVE** — ...
    #   **Verdict: APPROVE — ...
    #   ## Verdict\nAPPROVE — ...
    #   ## Verdict\n**REQUEST CHANGES** — ...
    # Strategy: grab the line containing "Verdict" plus the next 3 lines, then
    # match the first CLOSE / REQUEST CHANGES / NEEDS DISCUSSION / APPROVE keyword.
    has_close=0; has_rc=0; has_nd=0; has_ap=0
    models=""
    for f in "$latest_round"/*.md; do
      [[ -s "$f" ]] || continue
      # `|| true` because empty or no-match grep is fine under pipefail.
      # First try strict heading/bold patterns; fall back to any "verdict"
      # occurrence if not found (body text using "verdict" loosely is rare).
      ctx=$( { grep -A3 -E "^## Verdict|^\*\*Verdict" "$f" 2>/dev/null || true; } )
      [[ -z "$ctx" ]] && ctx=$( { grep -A3 -i "verdict" "$f" 2>/dev/null || true; } )
      v=$( printf '%s\n' "$ctx" | { grep -oE "REQUEST CHANGES|NEEDS DISCUSSION|APPROVE|CLOSE" || true; } | head -1)
      case "$v" in
        "CLOSE")             has_close=1 ;;
        "REQUEST CHANGES")   has_rc=1 ;;
        "NEEDS DISCUSSION")  has_nd=1 ;;
        "APPROVE")           has_ap=1 ;;
      esac

      # Parse model from filename: <model>_<reviewer>[__<tier>].md
      fname=$(basename "$f" .md)
      # strip optional __<tier> suffix, then take everything before the last _
      base_no_tier="${fname%%__*}"
      model="${base_no_tier%_*}"
      # strip leading "claude-" for compactness (e.g. claude-opus-4-7 → opus-4-7)
      model="${model#claude-}"
      [[ -z "$model" ]] && continue
      # append if not already in the list
      if [[ ",$models," != *",$model,"* ]]; then
        models="${models:+$models,}$model"
      fi
    done
    if   (( has_close )); then LOCAL_VERDICT[$n]="🚫 CLOSE"
    elif (( has_rc ));    then LOCAL_VERDICT[$n]="🔴 REQUEST CHANGES"
    elif (( has_nd ));    then LOCAL_VERDICT[$n]="🟡 NEEDS DISCUSSION"
    elif (( has_ap ));    then LOCAL_VERDICT[$n]="🟢 APPROVE"
    fi
    [[ -n "$models" ]] && LOCAL_MODELS[$n]="$models"
  done < <(find reviews/pr -mindepth 2 -maxdepth 2 -type d | sort)

  # jq computes one TSV row per non-draft open PR with: sort_key, n, title, author, url, state_icon, state_label
  jq -r --arg team "$TEAM_MEMBERS" '
    ($team | split(" ")) as $T
    | map(select(.isDraft | not))
    | map(
        . as $pr
        | ($pr.reviews // []) as $revs
        | ($pr.comments // []) as $comments
        # latest formal review state per team member (APPROVED / CHANGES_REQUESTED / COMMENTED)
        | (
            $T | map(
              . as $m
              | ($revs | map(select(.author.login == $m and (.state == "APPROVED" or .state == "CHANGES_REQUESTED" or .state == "COMMENTED"))) | sort_by(.submittedAt) | last // null)
              | select(. != null)
              | .state
            )
          ) as $member_states
        | (any($comments[]; .author.login as $a | $T | index($a))) as $any_comment
        | (
            if   ($member_states | index("CHANGES_REQUESTED")) then {icon: "🔴", label: "changes requested", rank: 2}
            elif ($member_states | index("APPROVED"))          then {icon: "🟢", label: "approved",          rank: 3}
            elif ($member_states | index("COMMENTED")) or $any_comment then {icon: "💬", label: "commented",  rank: 1}
            else                                                    {icon: "⏳", label: "no team review",   rank: 0}
            end
          ) as $state
        # which team members touched it (review or comment), de-duped
        | (
            ($revs    | map(select(.author.login as $a | $T | index($a)) | .author.login))
            +
            ($comments | map(select(.author.login as $a | $T | index($a)) | .author.login))
            | unique | join(",")
          ) as $touchers
        | {n: .number, title: .title, author: (.author.login // "?"), url: .url, updatedAt: .updatedAt, icon: $state.icon, label: $state.label, rank: $state.rank, touchers: $touchers}
      )
    | sort_by([.rank, ([(- (.updatedAt | fromdateiso8601 // 0))])])
    | .[]
    | [.rank, .n, .title, .author, .url, .icon, .label, .touchers] | @tsv
  ' "$TMP/open-prs.json" | while IFS=$'\t' read -r rank n title author url icon label touchers; do
    safe_title=$(printf '%s' "$title" | sed 's/|/\\|/g')
    if [[ -n "$touchers" ]]; then
      coverage="$icon $label ($touchers)"
    else
      coverage="$icon $label"
    fi
    local_rev="${LOCAL_REVIEW[$n]:-}"
    if [[ -n "$local_rev" ]]; then
      local_verdict="${LOCAL_VERDICT[$n]:-⚪ no verdict}"
      local_models="${LOCAL_MODELS[$n]:-}"
      if [[ -n "$local_models" ]]; then
        local_link="[$local_verdict · $local_models]($local_rev/)"
      else
        local_link="[$local_verdict]($local_rev/)"
      fi
    else
      local_link="—"
    fi
    echo "| [#$n]($url) | $safe_title | $author | $coverage | $local_link |"
  done

  echo

  # ── Team-authored open PRs needing iteration ────────────────────────────────
  # Focused view: open non-draft PRs authored by a Samourai team member where
  # our AI review verdict is 🔴 REQUEST CHANGES or 🟡 NEEDS DISCUSSION.
  # Reuses the LOCAL_VERDICT/LOCAL_REVIEW/LOCAL_MODELS maps built above.
  rc_lines=""
  nd_lines=""
  while IFS=$'\t' read -r n title author url; do
    local_verdict="${LOCAL_VERDICT[$n]:-}"
    [[ "$local_verdict" == "🔴 REQUEST CHANGES" || "$local_verdict" == "🟡 NEEDS DISCUSSION" ]] || continue
    local_rev="${LOCAL_REVIEW[$n]:-}"
    local_models="${LOCAL_MODELS[$n]:-}"
    safe_title=$(printf '%s' "$title" | sed 's/|/\\|/g')
    if [[ -n "$local_models" ]]; then
      link="[$local_verdict · $local_models]($local_rev/)"
    else
      link="[$local_verdict]($local_rev/)"
    fi
    row="| [#$n]($url) | $safe_title | $author | $link |"
    if [[ "$local_verdict" == "🔴 REQUEST CHANGES" ]]; then
      rc_lines+="$row"$'\n'
    else
      nd_lines+="$row"$'\n'
    fi
  done < <(jq -r --arg team "$TEAM_MEMBERS" '
    ($team | split(" ")) as $T
    | map(select(.isDraft | not))
    | map(select(.author.login as $a | $T | index($a)))
    | sort_by(- (.updatedAt | fromdateiso8601 // 0))
    | .[]
    | [.number, .title, (.author.login // "?"), .url] | @tsv
  ' "$TMP/open-prs.json")

  rc_count=$(printf '%s' "$rc_lines" | grep -c '^|' || true)
  nd_count=$(printf '%s' "$nd_lines" | grep -c '^|' || true)
  iter_total=$((rc_count + nd_count))

  echo "## Team-authored open PRs needing iteration ($iter_total)"
  echo
  echo "Open non-draft PRs authored by Samourai team members where our AI review flagged 🔴 REQUEST CHANGES or 🟡 NEEDS DISCUSSION. Sorted: 🔴 first ($rc_count), then 🟡 ($nd_count); within each, most-recently-updated first."
  echo
  if (( iter_total == 0 )); then
    echo "_None — all team-authored open PRs are clear of AI-flagged issues._"
  else
    echo "| PR | Title | Author | AI review |"
    echo "|---:|:------|:-------|:----------|"
    printf '%s' "$rc_lines"
    printf '%s' "$nd_lines"
  fi
  echo

  echo "## PR reviews"
  echo

  # Group by bucket dir name (newest bucket first)
  for bucket in $(find reviews/pr -mindepth 1 -maxdepth 1 -type d | sort -r); do
    bname=$(basename "$bucket")
    count=$(find "$bucket" -mindepth 1 -maxdepth 1 -type d | wc -l)
    echo "### \`$bname/\` ($count)"
    echo
    echo "| PR | Status | Title | Rounds | Reviewers |"
    echo "|---:|:------:|:------|:------:|:----------|"

    # Sort PR dirs by number desc
    while IFS= read -r d; do
      base=$(basename "$d")
      num="${base%%-*}"
      [[ "$num" =~ ^[0-9]+$ ]] || continue
      meta="$TMP/$num.json"
      [[ -f "$meta" ]] || continue

      state=$(jq -r '.state' "$meta")
      title=$(jq -r '.title' "$meta")
      url=$(jq -r '.url' "$meta")
      isDraft=$(jq -r '.isDraft // false' "$meta")
      [[ -z "$title" ]] && title=$(echo "$base" | sed "s/^${num}-//; s/-/ /g")

      case "$state" in
        OPEN)   icon=$([[ "$isDraft" == "true" ]] && echo "🟡" || echo "🟢") ;;
        MERGED) icon="🟣" ;;
        CLOSED) icon="🔴" ;;
        *)      icon="⚪" ;;
      esac

      # Count review rounds: subdirs matching <n>-<hash>
      rounds=$(find "$d" -mindepth 1 -maxdepth 1 -type d -regextype posix-extended -regex '.*/[0-9]+-[a-f0-9]+' | wc -l)

      # Reviewers: collect <model>_<reviewer>.md basenames across all rounds (unique reviewers)
      # Filename: <model>_<reviewer>[__<tier>].md — strip the optional __<tier>
      # suffix first, then capture the segment after the last underscore.
      reviewers=$(find "$d" -mindepth 2 -maxdepth 2 -type f -name '*.md' \
        | sed -E 's|__[^/]+\.md$|.md|' \
        | sed -E 's|.*_([^/_]+)\.md$|\1|' \
        | sort -u | paste -sd ', ' -)
      [[ -z "$reviewers" ]] && reviewers="—"

      # Escape pipes in title
      safe_title=$(printf '%s' "$title" | sed 's/|/\\|/g')

      rel="${d#reviews/}"
      echo "| [#$num]($url) | $icon | [$safe_title]($rel/) | $rounds | $reviewers |"
    done < <(find "$bucket" -mindepth 1 -maxdepth 1 -type d | sort -t/ -k4 -r)
    echo
  done

  # Security
  if [[ -d reviews/security ]]; then
    sec_count=$(find reviews/security -mindepth 1 -maxdepth 1 -type d | wc -l)
    echo "## Security reports ($sec_count)"
    echo
    echo "HackenProof reports under [\`reviews/security/\`](security/). Status tracked in HackenProof, not here."
    echo
    echo "| ID | Slug |"
    echo "|----|------|"
    find reviews/security -mindepth 1 -maxdepth 1 -type d | sort | while IFS= read -r d; do
      base=$(basename "$d")
      id=$(echo "$base" | grep -oE '^[A-Z]+-[0-9]+' || true)
      [[ -z "$id" ]] && id="$base"
      slug="${base#${id}-}"
      [[ "$slug" == "$base" ]] && slug=""
      echo "| \`$id\` | [$slug](security/$base/) |"
    done
    echo
  fi
} > "$BODY"

# Assemble: header + legends + auto-TOC + body.
{
  echo "# Reviews"
  echo
  echo "PR reviews of [gnolang/gno](https://github.com/$REPO), bucketed by PR number range. Auto-generated by [\`scripts/build-reviews-readme.sh\`](../scripts/build-reviews-readme.sh) — do not edit by hand."
  echo
  echo "_Last updated: $(date -u +%Y-%m-%dT%H:%M:%SZ)_"
  echo
  echo "Status legend: 🟢 open · 🟡 draft · 🟣 merged · 🔴 closed · ⚪ unknown"
  echo
  echo "Team coverage legend: 🟢 approved · 🔴 changes requested · 💬 commented only · ⏳ no team review"
  echo
  echo "AI review verdict (parsed from our AI-generated review files): 🟢 APPROVE · 🔴 REQUEST CHANGES · 🟡 NEEDS DISCUSSION · 🚫 CLOSE (PR should not be merged at all — superseded, abandoned, or wrong direction) · ⚪ no verdict (file exists but verdict line was not parseable). Model used follows after \`·\` (e.g. \`opus-4-7\`; \`claude-\` prefix stripped)."
  echo

  # ── Table of contents ───────────────────────────────────────────────────────
  # Build entries from real H2/H3 headings in the body. Slug = GitHub-flavored:
  # lowercase, drop backticks, strip everything but [a-z0-9 _-], collapse and
  # trim spaces, spaces→hyphens.
  echo "## Contents"
  echo
  while IFS= read -r line; do
    case "$line" in
      "## "*)  title="${line#"## "}";  indent="" ;;
      "### "*) title="${line#"### "}"; indent="  " ;;
      *) continue ;;
    esac
    slug=$(printf '%s' "$title" \
      | tr '[:upper:]' '[:lower:]' \
      | sed -e 's/`//g' -e 's/[^a-z0-9 _-]//g' -e 's/  */ /g' -e 's/^ //; s/ $//' -e 's/ /-/g')
    echo "${indent}- [${title}](#${slug})"
  done < <(grep -E '^##+ ' "$BODY")
  echo

  cat "$BODY"
} > "$OUTPUT"

echo "wrote $OUTPUT" >&2
