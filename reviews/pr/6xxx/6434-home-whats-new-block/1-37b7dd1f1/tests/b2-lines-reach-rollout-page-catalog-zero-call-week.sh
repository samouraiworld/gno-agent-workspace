#!/usr/bin/env bash
#
# A week with no realm call fails source_used instead of counting zero, so the
# --builders page keeps last week's "active users this week" and ranking.
#
# Repro from a plain clone of gnolang/gno:
#   git fetch origin pull/6434/head && git checkout 37b7dd1f15a79144153303fe49a98e71f6ff9846
#   WHATSNEW=$PWD/misc/deployments/home-alias/whatsnew.sh bash <this file>
#
# Same harness as misc/deployments/home-alias/whatsnew_test.sh: the script is
# sourced and graphql() stubbed. source_used runs under `set -euo pipefail`,
# which is how `whatsnew.sh --source used` runs it from refresh().
set -uo pipefail
script=${WHATSNEW:?set WHATSNEW to the path of whatsnew.sh}
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
export STATE_DIR=$work/state WHATSNEW_SCRIPT=$script
mkdir -p "$STATE_DIR"

one='[{"block_height":10,"success":true,"messages":[{"value":{"__typename":"MsgCall","caller":"g1caller","pkg_path":"gno.land/r/demo/foo"}}]}]'

run_used() {
  CALLS=$1 USED_NAMESPACES= bash -c '
    source "$WHATSNEW_SCRIPT"
    graphql() {
      case $1 in
      *latestBlockHeight*) echo "{\"data\":{\"latestBlockHeight\":500}}" ;;
      *getBlocks*) echo "{\"data\":{\"getBlocks\":[{\"time\":\"2026-11-19T00:00:00Z\"}]}}" ;;
      *getTransactions*) echo "{\"data\":{\"getTransactions\":$CALLS}}" ;;
      esac
    }
    source_used'
}

# Yesterday's run: one caller of one realm.
run_used "$one" >"$STATE_DIR/used"
echo "week with one call: rc=$? activity=[$(tr '\t' ' ' <"$STATE_DIR/activity")]"
touch -d '1 day ago' "$STATE_DIR/used" "$STATE_DIR/activity"

# Today's run: the indexer answers, the week holds no call.
out=$(run_used '[]' 2>&1)
echo "week with no call:  rc=$? out=[$out] activity=[$(tr '\t' ' ' <"$STATE_DIR/activity")] mtime=$(stat -c %y "$STATE_DIR/activity" | cut -c1-16)"

# A week with no deployment writes its zero, for contrast.
echo 0 >"$STATE_DIR/deployed-count"

# What the page shows.
bash -c 'source "$WHATSNEW_SCRIPT"; render_builders' | grep -E 'this week|weekly users|^1\. '
