#!/usr/bin/env bash
# b2-rollout: gnolang/gno#6298 at 065ec369b, two binaries at once.
#
# The button is rendered by gnoweb (ExtButtons.Extend, gno.land/pkg/gnoweb/markdown/ext.go:96,
# unconditional) but the escape that keeps sanitized user content from producing one lives in
# the chain/markdown VM native, run by the RPC node that executes Render. gnoweb is its own
# image and a patch-level release; the native only changes at a coordinated halt. This script
# pairs the chain/markdown native gnoland-1 runs today (identical to the merge base at every
# upgrades.json commit, v1.2.0..v1.5.0) with head's gnoweb, through the repo's own sanitize
# golden harness (TestSanitizeIntegration), and prints:
#   1. how many sanitize goldens change their native output bytes (output.md), i.e. the bytes a
#      MsgRun printing sanitize.Block(x) returns as Data differ between the two binaries;
#   2. how many of them then render a live <a class="gno-button ..."> in output.html.
#
# Repro from a plain clone:
#   git clone https://github.com/gnolang/gno gno && cd gno
#   git fetch origin pull/6298/head
#   git checkout --detach 065ec369b321c207815f82b0c1dbf2b0f9f10b01
#   bash /path/to/b2-rollout-old-native-new-gnoweb.sh      # Go 1.25.x on PATH
# The script edits the tree and restores it with git checkout at the end.
#
# Observed on 2026-10-08 (go1.25.9 linux/amd64):
#   mainnet native == base: 9c8eb132e4 31b6650a10 00417a1be9 e75fef82c0 (all four)
#   goldens whose output.md changes under the mainnet native: 30
#   goldens rendering a live gno-button under the mainnet native: 28
#   sample: <p>hi <a href="/r/evil$help&amp;func=Drain" class="gno-button gno-button-caution">Claim...
set -euo pipefail

BASE=b0656512d971e26e4e08f3d4bfa736a5bf5ba635
NATIVE=gnovm/stdlibs/chain/markdown/markdown.go
GOLDEN=gno.land/pkg/gnoweb/markdown/golden/sanitize

restore() { git checkout -- "$NATIVE" "$GOLDEN"; }
trap restore EXIT

printf 'mainnet native == base:'
for c in $(python3 -c "import json;print(' '.join(u['commit'] for u in json.load(open('misc/deployments/mainnet.gno.land/upgrades.json'))['upgrades']))"); do
	if git cat-file -e "$c" 2>/dev/null; then
		git diff --quiet "$c" "$BASE" -- "$NATIVE" && printf ' %s' "${c:0:10}" || printf ' %s(differs)' "${c:0:10}"
	else
		printf ' %s(not fetched)' "${c:0:10}"
	fi
done
echo

# Sanity: head passes its own goldens.
go test ./gno.land/pkg/gnoweb/markdown -run TestSanitizeIntegration -count=1 >/dev/null

# Node on the old binary, gnoweb on head.
git show "$BASE:$NATIVE" >"$NATIVE"
go test ./gno.land/pkg/gnoweb/markdown -run TestSanitizeIntegration -count=1 -update-golden-tests >/dev/null

echo "goldens whose output.md changes under the mainnet native: $(git diff --name-only -- "$GOLDEN" | wc -l)"
echo "goldens rendering a live gno-button under the mainnet native: $(git diff -U0 -- "$GOLDEN" | grep -c '^+.*class="gno-button')"
echo "sample:"
git diff -U0 -- "$GOLDEN/block-gno-button-midline-escaped.txtar" | grep '^+<p>' | cut -c2-140
