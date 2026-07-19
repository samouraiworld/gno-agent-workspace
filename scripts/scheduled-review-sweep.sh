#!/usr/bin/env bash
# NOT AUDITED — AI-generated tooling. Review before executing in any privileged context.
#
# scheduled-review-sweep.sh
#
# Autonomous PR review sweep. Reads a TSV list of PR numbers and dispatches
# `gno-review` via headless Claude Code (`claude -p`) in batches. Cleans up
# worktrees between batches to bound disk usage. At the end, rebuilds the
# reviews/README.md index and pushes the batch.
#
# Designed to run unattended from systemd-run / cron / at — every step logs
# to a timestamped file under data/scheduled/.
#
# Usage:
#   ./scripts/scheduled-review-sweep.sh [<pr-list.tsv>] [<batch-size>]
# Defaults:
#   pr-list.tsv  = data/scheduled/pending-reviews-<today>.tsv
#   batch-size   = 6

set -uo pipefail

# Workspace root. Defaults to this script's parent directory; override with
# GNO_WORKSPACE_DIR when the checkout lives elsewhere.
WORK_DIR="${GNO_WORKSPACE_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
cd "$WORK_DIR"

# Single-instance guard: refuse to run if another sweep already holds the lock.
exec 9>/tmp/gno-review-sweep.lock
if ! flock -n 9; then
  echo "another sweep is already running (lock /tmp/gno-review-sweep.lock held); exiting" >&2
  exit 1
fi

LIST="${1:-$WORK_DIR/data/scheduled/pending-reviews-$(date +%Y-%m-%d).tsv}"
BATCH_SIZE="${2:-6}"

if [[ ! -s "$LIST" ]]; then
  echo "list not found or empty: $LIST" >&2
  exit 1
fi

mkdir -p "$WORK_DIR/data/scheduled"
LOG="$WORK_DIR/data/scheduled/sweep-$(date +%Y%m%d-%H%M%S).log"
exec > >(tee -a "$LOG") 2>&1

echo "[$(date -u +%FT%TZ)] === Sweep start ==="
echo "List:       $LIST"
echo "Batch size: $BATCH_SIZE"
echo "Log:        $LOG"

mapfile -t PRS < <(awk -F'\t' '{print $1}' "$LIST" | grep -E '^[0-9]+$')
echo "Total PRs:  ${#PRS[@]}"

if (( ${#PRS[@]} == 0 )); then
  echo "no PRs to process"
  exit 0
fi

# Fetch gno master once up front so per-PR worktree creation is fast.
git -C "$WORK_DIR/gno" fetch origin master 2>&1 | tail -3 || true

cleanup_worktrees() {
  local prs=("$@")
  for pr in "${prs[@]}"; do
    local wt="$WORK_DIR/.worktrees/gno-review-$pr"
    [[ -d "$wt" ]] || continue
    git -C "$WORK_DIR/gno" worktree remove --force "$wt" 2>/dev/null || rm -rf "$wt"
  done
  git -C "$WORK_DIR/gno" worktree prune 2>/dev/null || true
}

# Batch loop
total_done=0
for ((i=0; i<${#PRS[@]}; i+=BATCH_SIZE)); do
  BATCH=("${PRS[@]:i:BATCH_SIZE}")
  PR_LIST_SPACE="${BATCH[*]}"

  echo
  echo "[$(date -u +%FT%TZ)] === Batch $((i/BATCH_SIZE + 1)): $PR_LIST_SPACE ==="

  # Pre-batch disk check: refuse if we can't fit this batch's worktrees.
  # Empirical: ~200 MB per worktree, so reserve BATCH_SIZE * 250 MB plus 1 GB headroom.
  AVAIL_MB=$(df --output=avail / | tail -1 | awk '{print int($1/1024)}')
  NEED_MB=$(( ${#BATCH[@]} * 250 + 1024 ))
  echo "[$(date -u +%FT%TZ)] Free disk: $((AVAIL_MB/1024))G ; need ~$((NEED_MB/1024))G for this batch"
  if (( AVAIL_MB < NEED_MB )); then
    echo "[$(date -u +%FT%TZ)] ABORT: free disk ($((AVAIL_MB/1024))G) below needed ($((NEED_MB/1024))G); stopping early"
    break
  fi

  # Hand the batch to a fresh headless claude.  The skill at skills/review.md
  # already documents parallel dispatch when given more than one PR number.
  PROMPT=$(cat <<EOF
You are running a scheduled, unattended PR-review sweep. Run the workflow at \`skills/review.md\` for these PRs: $PR_LIST_SPACE. Dispatch each PR as a parallel Agent in a single message (no sequencing). Use \`subagent_type: general-purpose\`. Do NOT commit or push — the parent wrapper does that at the end of the whole sweep. When all subagents return, exit cleanly. Report back only a one-line summary per PR: "#<N>: <verdict>".
EOF
)

  # --dangerously-skip-permissions: no human to approve; the workflow is well-known.
  # --add-dir: explicit access to the workspace and to the gno checkout.
  claude --dangerously-skip-permissions \
         --add-dir "$WORK_DIR" \
         --add-dir "$WORK_DIR/gno" \
         -p "$PROMPT" 2>&1 | tee -a "$LOG.batch.$((i/BATCH_SIZE + 1))" || {
    echo "[$(date -u +%FT%TZ)] WARN: batch claude call returned non-zero; continuing"
  }

  total_done=$((total_done + ${#BATCH[@]}))
  echo "[$(date -u +%FT%TZ)] Batch done ($total_done/${#PRS[@]}); cleaning worktrees"
  cleanup_worktrees "${BATCH[@]}"
done

# Final: rebuild index, commit + push. Push is gated behind the main-branch check.
#
# ⚠️  NEVER USE --force / --force-with-lease ON THIS PUSH. Reviews are append-only
#     artifacts on the main branch; a force-push could overwrite teammates' commits.
#     If you find yourself needing --force here, something else is wrong upstream.
echo
echo "[$(date -u +%FT%TZ)] === Final: rebuild + commit + push ==="
./scripts/build-reviews-readme.sh
COUNT=$(git status --porcelain reviews/ | wc -l)
if (( COUNT > 0 )); then
  BRANCH=$(git rev-parse --abbrev-ref HEAD)
  if [[ "$BRANCH" != "main" ]]; then
    echo "[$(date -u +%FT%TZ)] REFUSE commit+push: HEAD is on '$BRANCH', expected 'main'. Reviews left uncommitted in working tree."
    PUSH_HINT="checkout main, then commit + push manually"
  else
    git add reviews/ docs/glossary.md
    git commit -m "review: scheduled sweep — $total_done PRs

Autonomous review sweep dispatched via scheduled-review-sweep.sh.
List: $(basename "$LIST")
Log:  $(basename "$LOG")"
    if git push 2>&1; then
      echo "[$(date -u +%FT%TZ)] Committed and pushed."
      PUSH_HINT="pushed to origin"
    else
      echo "[$(date -u +%FT%TZ)] Push failed; commit is local — run \`git push\` manually."
      PUSH_HINT="commit landed locally; push failed, run 'git push' manually"
    fi
  fi
else
  echo "[$(date -u +%FT%TZ)] No changes to commit."
  PUSH_HINT="nothing to push"
fi

# Final notification
notify-send -u normal -t 0 "Gno PR review sweep done" \
  "$total_done/${#PRS[@]} reviewed. $PUSH_HINT. Log: $LOG" 2>/dev/null || true

echo "[$(date -u +%FT%TZ)] === Sweep end ==="
