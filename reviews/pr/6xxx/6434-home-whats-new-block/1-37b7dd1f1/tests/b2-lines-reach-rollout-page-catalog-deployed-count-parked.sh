#!/usr/bin/env bash
#
# "N deployments this week" on the --builders page counts every successful
# MsgAddPackage, and onyx runs code_submission_policy "inert": a submission is
# parked, not online, until it is enabled. The list under it keeps only the
# live ones (vm/qfile), so the count and the list disagree.
#
# Live and read-only, against onyx's public RPC and tx-indexer. Repro from a
# plain clone of gnolang/gno:
#   git fetch origin pull/6434/head && git checkout 37b7dd1f15a79144153303fe49a98e71f6ff9846
#   WHATSNEW=$PWD/misc/deployments/home-alias/whatsnew.sh bash <this file>
set -uo pipefail
script=${WHATSNEW:?set WHATSNEW to the path of whatsnew.sh}
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
export STATE_DIR=$work WHATSNEW_SCRIPT=$script
export RPC_URL=https://rpc.onyx.testnets.gno.land:443
export INDEXER_URL=https://indexer.onyx.testnets.gno.land/graphql/query

curl -fsSL --max-time 20 -G --data-urlencode 'path="params/vm:p:code_submission_policy"' "$RPC_URL/abci_query" |
  jq -r '"code_submission_policy: " + (.result.response.ResponseBase.Data | @base64d)'

# What the page prints: the count, then the list.
bash "$script" --source deployed >"$work/deployed"
echo "source deployed rc=$?"
echo "deployments this week (page): $(cat "$work/deployed-count")"
echo "Recently deployed rows (page): $(grep -c . "$work/deployed")"

# Every deployment the count holds, each checked the way live_rows checks one.
bash -c '
  source "$WHATSNEW_SCRIPT"
  scan_week "MsgAddPackage: { package: { path: { like: \".\" } } }" "... on MsgAddPackage { package { path } }" \
    "select(.__typename == \"MsgAddPackage\") | [\$height, .package.path] | @tsv" |
    awk -F"\t" -v skip="$DEPLOYED_SKIP" "/^[0-9]+\tgno\.land\/[rp]\/[a-z0-9_.\/-]+\$/ && !index(\$0, \"..\") && (skip == \"\" || \$2 !~ skip)" |
    cut -f2 | while read -r pkg; do
      if abci "$RPC_URL" vm/qfile "$pkg" >/dev/null 2>&1; then echo live; else echo "not-live $pkg"; fi
    done' >"$work/checked"
echo "counted deployments online now: $(grep -c '^live$' "$work/checked") of $(grep -c . "$work/checked")"
grep '^not-live' "$work/checked" | head -5
