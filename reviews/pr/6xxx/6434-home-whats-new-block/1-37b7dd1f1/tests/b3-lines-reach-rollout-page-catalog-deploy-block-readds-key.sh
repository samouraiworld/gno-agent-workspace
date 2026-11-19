#!/usr/bin/env bash
#
# PR 6434, head 37b7dd1f1: the testnet home's "Deploy from your terminal" prose
# has the visitor add a key and fund it, then the block under it opens on
# `gnokey add alice`. On an existing "alice" that line asks
# `Override the existing key "alice" [Y/n]:`, whose default is yes, so the
# Enter a visitor presses replaces the funded key with a new, unfunded one.
# whatsnew_test.sh stubs gnokey as `exit 0`, so its deploy-block check passes.
#
# Repro from a plain clone (needs go and script(1) from util-linux):
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin pull/6434/head && git checkout --detach 37b7dd1f15a79144153303fe49a98e71f6ff9846
#   bash <path to this file>
# Expect: "FUNDED KEY REPLACED" and exit 1 at this head.
set -euo pipefail

root=$(git rev-parse --show-toplevel)
page=$root/misc/deployments/home-alias/home.testnet.md
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# The block's first line, extracted the way whatsnew_test.sh extracts the block.
first=$(awk '/^## .*Deploy from your terminal/ { on = 1 } on && /^```sh$/ { inside = 1; next } inside && /^```$/ { exit } inside' "$page" | head -n 1)
echo "block line 1: $first"
[ "$first" = "gnokey add alice" ] || { echo "block no longer opens on gnokey add alice"; exit 0; }

(cd "$root" && go build -o "$tmp/gnokey" ./gno.land/cmd/gnokey)
home=$tmp/home

# One Enter a second, on a terminal, as a visitor answers the prompts.
enter() {
  (for _ in 1 2 3 4; do sleep 1; printf '\r'; done) |
    script -qec "$tmp/gnokey add alice -home $home" /dev/null >/dev/null 2>&1 || true
}

enter # the prose: "Add a key ... and fund that address"
funded=$("$tmp/gnokey" list -home "$home" | grep -oE 'g1[0-9a-z]{38}')
enter # the block's first line
after=$("$tmp/gnokey" list -home "$home" | grep -oE 'g1[0-9a-z]{38}')
echo "funded: $funded"
echo "after:  $after"
if [ "$funded" != "$after" ]; then
  echo "FUNDED KEY REPLACED"
  exit 1
fi
echo "key kept"
